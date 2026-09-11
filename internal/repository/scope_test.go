package repository

import (
	"context"
	"strings"
	"testing"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"photography-server/internal/domain"
	"photography-server/internal/model"
)

// dryRunDB 构造一个不建立真实连接、只生成 SQL 的 GORM 实例。
// 数据权限的核心是「生成了什么 WHERE」，用 DryRun 断言可完全脱离数据库运行。
func dryRunDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(mysql.New(mysql.Config{
		DSN:                       "root:root@tcp(127.0.0.1:3306)/photography?charset=utf8&parseTime=True",
		SkipInitializeWithVersion: true,
	}), &gorm.Config{DryRun: true, Logger: logger.Discard})
	if err != nil {
		t.Fatalf("打开 DryRun 连接失败: %v", err)
	}
	return db
}

// scopeSQL 执行 DryRun 查询并取回生成的 SQL 与参数。
// dest 必须与 q 上 Model(...) 的类型一致，否则 GORM 会用 dest 的表名覆盖。
func scopeSQL(t *testing.T, q *gorm.DB, dest interface{}) (string, []interface{}) {
	t.Helper()
	tx := q.Find(dest)
	if tx.Error != nil {
		t.Fatalf("构建 SQL 失败: %v", tx.Error)
	}
	return tx.Statement.SQL.String(), tx.Statement.Vars
}

func hasVar(vars []interface{}, want interface{}) bool {
	for _, v := range vars {
		if v == want {
			return true
		}
	}
	return false
}

// TestApplyScope 锁定数据权限的三档语义与两条降级规则
// （任何一条被改坏都会导致越权或数据不可见，属安全关键路径）。
func TestApplyScope(t *testing.T) {
	db := dryRunDB(t)

	cases := []struct {
		name      string
		op        domain.Operator
		cols      ScopeCols
		wantInSQL []string
		wantNotIn []string
		wantVars  []interface{}
	}{
		{
			name:      "全部数据_不加任何条件",
			op:        domain.Operator{UserID: 9, StoreID: 7, DataScope: domain.ScopeAll},
			cols:      scopeOrder,
			wantNotIn: []string{"store_id = ?", "photographer_id = ?", "1 = 0"},
		},
		{
			name:      "零值_视为未启用权限_不加条件",
			op:        domain.Operator{UserID: 9, StoreID: 7},
			cols:      scopeOrder,
			wantNotIn: []string{"store_id = ?", "photographer_id = ?", "1 = 0"},
		},
		{
			name:      "本门店_按store_id过滤",
			op:        domain.Operator{UserID: 9, StoreID: 7, DataScope: domain.ScopeStore},
			cols:      scopeOrder,
			wantInSQL: []string{"store_id = ?"},
			wantVars:  []interface{}{int64(7)},
		},
		{
			name:      "本门店但未分配门店_降级为仅本人（防越过门店边界）",
			op:        domain.Operator{UserID: 9, StoreID: 0, DataScope: domain.ScopeStore},
			cols:      scopeOrder,
			wantInSQL: []string{"photographer_id = ?", "OR owner_id = ?"},
			wantNotIn: []string{"store_id = ?"},
			wantVars:  []interface{}{int64(9)},
		},
		{
			name:      "仅本人_多列取OR",
			op:        domain.Operator{UserID: 9, StoreID: 7, DataScope: domain.ScopeSelf},
			cols:      scopeOrder,
			wantInSQL: []string{"photographer_id = ? OR owner_id = ?"},
			wantNotIn: []string{"store_id = ?"},
			wantVars:  []interface{}{int64(9)},
		},
		{
			name:      "共享资源_仅本人降级为本门店（套餐）",
			op:        domain.Operator{UserID: 9, StoreID: 7, DataScope: domain.ScopeSelf},
			cols:      scopePackage,
			wantInSQL: []string{"store_id = ?"},
			wantNotIn: []string{"1 = 0"},
			wantVars:  []interface{}{int64(7)},
		},
		{
			name:      "无归属人列_仅本人时查不到数据（安全兜底）",
			op:        domain.Operator{UserID: 9, DataScope: domain.ScopeSelf},
			cols:      ScopeCols{Store: "store_id"},
			wantInSQL: []string{"1 = 0"},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var out []model.Order
			sql, vars := scopeSQL(t,
				applyScope(db.Model(&model.Order{}).Where("company_id = ?", int64(100)), c.op, c.cols),
				&out)
			for _, want := range c.wantInSQL {
				if !strings.Contains(sql, want) {
					t.Errorf("SQL 应包含 %q，实际: %s", want, sql)
				}
			}
			for _, notWant := range c.wantNotIn {
				if strings.Contains(sql, notWant) {
					t.Errorf("SQL 不应包含 %q，实际: %s", notWant, sql)
				}
			}
			for _, v := range c.wantVars {
				if !hasVar(vars, v) {
					t.Errorf("参数应包含 %v，实际: %v", v, vars)
				}
			}
			// 多租户条件任何情况下都必须在（数据权限不能替代租户隔离）
			if !strings.Contains(sql, "company_id = ?") || !hasVar(vars, int64(100)) {
				t.Errorf("租户条件丢失，实际: %s / %v", sql, vars)
			}
		})
	}
}

// TestApplyScopePublic 锁定「公共池」语义（定制需求：store_id=0 对员工可见可认领）。
func TestApplyScopePublic(t *testing.T) {
	db := dryRunDB(t)

	assertSQL := func(name, wantIn string, wantNotIn []string, wantVar interface{}) {
		t.Helper()
		var out []model.CustomRequest
		op := domain.Operator{UserID: 9, StoreID: 7, DataScope: domain.ScopeStore}
		if name == "仅本人_归属人OR公共池" {
			op.DataScope = domain.ScopeSelf
		}
		if name == "全部数据_不加条件" {
			op.DataScope = domain.ScopeAll
		}
		sql, vars := scopeSQL(t,
			applyScope(db.Model(&model.CustomRequest{}).Where("company_id = ?", int64(100)), op, scopeCustomRequest),
			&out)
		if wantIn != "" && !strings.Contains(sql, wantIn) {
			t.Errorf("[%s] SQL 应包含 %q，实际: %s", name, wantIn, sql)
		}
		for _, notWant := range wantNotIn {
			if strings.Contains(sql, notWant) {
				t.Errorf("[%s] SQL 不应包含 %q，实际: %s", name, notWant, sql)
			}
		}
		if wantVar != nil && !hasVar(vars, wantVar) {
			t.Errorf("[%s] 参数应包含 %v，实际: %v", name, wantVar, vars)
		}
	}

	// 本门店：本店 + 公共池（store_id=0）
	assertSQL("本门店_本店OR公共池", "(store_id = ? OR store_id = 0)", nil, int64(7))
	// 仅本人：我响应过的 + 公共池
	assertSQL("仅本人_归属人OR公共池", "(response_by = ? OR store_id = 0)", nil, int64(9))
	// 全部数据：不加任何条件
	assertSQL("全部数据_不加条件", "", []string{"store_id = ?"}, nil)
}

// TestScopedFromOrder 锁定「无 store_id 的从表」经可见订单传递过滤的行为。
func TestScopedFromOrder(t *testing.T) {
	db := dryRunDB(t)
	repo := &Repo{db: db} // 同包直连，避免依赖包级 root

	// 本门店：应追加 order_id IN (SELECT id FROM biz_order WHERE company_id = ? AND store_id = ?)
	ctx := domain.WithOperator(context.Background(), domain.Operator{StoreID: 7, DataScope: domain.ScopeStore})
	var pays []model.OrderPayment
	sql, vars := scopeSQL(t,
		repo.scopedFromOrder(db.Model(&model.OrderPayment{}).Where("company_id = ?", int64(100)), ctx, 100),
		&pays)
	// 生成的子查询形如：
	//   order_id IN (SELECT `id` FROM `biz_order` WHERE company_id = ? AND store_id = ? AND `biz_order`.`deleted` = ?)
	if !strings.Contains(sql, "order_id IN (SELECT `id` FROM `biz_order` WHERE company_id = ? AND store_id = ?") {
		t.Errorf("未生成可见订单子查询，实际: %s", sql)
	}
	if !hasVar(vars, int64(7)) {
		t.Errorf("子查询参数应含门店 7，实际: %v", vars)
	}

	// 全部数据：不应产生子查询（既保证行为不变，也省掉一次无谓查询）
	ctxAll := domain.WithOperator(context.Background(), domain.Operator{DataScope: domain.ScopeAll})
	sqlAll, _ := scopeSQL(t,
		repo.scopedFromOrder(db.Model(&model.OrderPayment{}).Where("company_id = ?", int64(100)), ctxAll, 100),
		&pays)
	if strings.Contains(sqlAll, "order_id IN") {
		t.Errorf("全部数据不应追加子查询，实际: %s", sqlAll)
	}

	// 未经认证的链路（客户侧 / 定时任务）：同样放行
	sqlAnon, _ := scopeSQL(t,
		repo.scopedFromOrder(db.Model(&model.OrderPayment{}).Where("company_id = ?", int64(100)), context.Background(), 100),
		&pays)
	if strings.Contains(sqlAnon, "order_id IN") {
		t.Errorf("无操作人上下文不应追加子查询，实际: %s", sqlAnon)
	}
}

// TestOperatorContext 锁定 context 存取往返（service → repository 的传递载体）。
func TestOperatorContext(t *testing.T) {
	if _, ok := domain.OperatorFrom(context.Background()); ok {
		t.Error("空 context 不应取到操作人")
	}
	want := domain.Operator{UserID: 9, CompanyID: 1, StoreID: 7, DataScope: domain.ScopeStore, Permissions: []string{"order:view"}}
	got, ok := domain.OperatorFrom(domain.WithOperator(context.Background(), want))
	if !ok {
		t.Fatal("应能取回操作人")
	}
	if got.UserID != want.UserID || got.CompanyID != want.CompanyID ||
		got.StoreID != want.StoreID || got.DataScope != want.DataScope ||
		len(got.Permissions) != 1 || got.Permissions[0] != "order:view" {
		t.Errorf("往返失败: got=%+v want=%+v", got, want)
	}
}
