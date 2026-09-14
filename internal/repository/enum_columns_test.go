package repository

import (
	"reflect"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"photography-server/internal/enum"
	"photography-server/internal/model"
)

// newDryRunDB 构造绑定 sqlmock 的 *gorm.DB，只用于 DryRun 取 SQL 与绑定参数（不真正执行）。
func newDryRunDB(t *testing.T) *gorm.DB {
	t.Helper()
	sqlDB, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New failed: %v", err)
	}
	t.Cleanup(func() { sqlDB.Close() })

	db, err := gorm.Open(mysql.New(mysql.Config{
		Conn:                      sqlDB,
		SkipInitializeWithVersion: true,
	}), &gorm.Config{SkipDefaultTransaction: true})
	if err != nil {
		t.Fatalf("gorm.Open failed: %v", err)
	}
	return db
}

// assertIntColumn 断言 DryRun 生成的 INSERT 中，指定列绑定的是整型值。
// 背景：MySQL 严格模式（STRICT_TRANS_TABLES）下，向 tinyint 列写字符串会被直接拒收：
//
//	ERROR 1366: Incorrect integer value: '图片' for column 'file_type' at row 1
//
// 已实测复现（.workbuddy/artifacts/上传接口报错-2026-09-14.md），故此处锁定绑定类型。
func assertIntColumn(t *testing.T, stmt *gorm.Statement, column string, want int64) {
	t.Helper()
	sql := stmt.SQL.String()

	open := strings.Index(sql, "(")
	end := strings.Index(sql, ") VALUES")
	if open < 0 || end < open {
		t.Fatalf("无法解析 INSERT 列清单: %s", sql)
	}
	idx := -1
	for i, c := range strings.Split(sql[open+1:end], ",") {
		if strings.Trim(strings.TrimSpace(c), "`") == column {
			idx = i
			break
		}
	}
	if idx < 0 {
		t.Fatalf("INSERT 未包含列 %s: %s", column, sql)
	}
	if idx >= len(stmt.Vars) {
		t.Fatalf("列 %s 下标 %d 越界（Vars 长度 %d）: %s", column, idx, len(stmt.Vars), sql)
	}

	got := stmt.Vars[idx]
	rv := reflect.ValueOf(got)
	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if rv.Int() != want {
			t.Errorf("列 %s 绑定值 = %d, want %d", column, rv.Int(), want)
		}
	default:
		t.Fatalf("列 %s 绑定值 = %#v（%T），必须为整型；写字符串会被 MySQL 严格模式下的 tinyint 列拒收",
			column, got, got)
	}
}

// TestEnumColumnsBindAsInt 回归（2026-09-14 /upload/file 报错）：
// 以下 tinyint 列历史上都用 string 承载文本值（"图片"/"sample"/"order"），
// 在 MySQL 严格模式下写入即报 1366，且通知写入失败还被 logger.Warnf 吞掉。
// 该用例锁定这几列绑定的必须是整型枚举。
func TestEnumColumnsBindAsInt(t *testing.T) {
	db := newDryRunDB(t)

	cases := []struct {
		name   string
		entity interface{}
		column string
		want   int64
	}{
		{"biz_upload.file_type", &model.Upload{
			BizType:  "order",
			FileType: enum.UploadTypeImage,
			FileURL:  "/uploads/202609/ab12cd34.jpg",
		}, "file_type", int64(enum.UploadTypeImage)},

		{"biz_delivery_item.file_type", &model.DeliveryItem{
			DeliveryID: 1,
			URL:        "/uploads/202609/ab12cd34.mp4",
			FileType:   enum.UploadTypeVideo,
		}, "file_type", int64(enum.UploadTypeVideo)},

		{"biz_delivery_item.kind", &model.DeliveryItem{
			DeliveryID: 1,
			URL:        "/uploads/202609/ab12cd34.jpg",
			Kind:       enum.DeliveryItemKindRetouched,
		}, "kind", int64(enum.DeliveryItemKindRetouched)},

		{"sys_notification.type", &model.SysNotification{
			ReceiverID:   1,
			ReceiverType: enum.NotificationReceiverStaff,
			Type:         enum.NotificationTypeFinance,
			Title:        "收款已确认到账",
		}, "type", int64(enum.NotificationTypeFinance)},

		// biz_order_log.from_status / to_status：DDL 为 tinyint，历史上模型是 string 并
		// 用 fmt.Sprintf("%v", from) 把枚举转成 "1" 这类数字串，靠 MySQL 隐式转换侥幸入库。
		// 现改为 int，此处锁定其绑定值必须是整型。
		{"biz_order_log.from_status", &model.OrderLog{
			OrderID:    1,
			Action:     "change_status",
			FromStatus: int(enum.OrderStatusPendingDeposit),
			ToStatus:   int(enum.OrderStatusPendingShoot),
		}, "from_status", int64(enum.OrderStatusPendingDeposit)},

		{"biz_order_log.to_status", &model.OrderLog{
			OrderID:    1,
			Action:     "change_status",
			FromStatus: int(enum.OrderStatusPendingDeposit),
			ToStatus:   int(enum.OrderStatusPendingShoot),
		}, "to_status", int64(enum.OrderStatusPendingShoot)},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			stmt := db.Session(&gorm.Session{DryRun: true}).Create(c.entity).Statement
			assertIntColumn(t, stmt, c.column, c.want)
		})
	}
}
