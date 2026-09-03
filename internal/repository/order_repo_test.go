package repository

import (
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"photography-server/internal/enum"
)

// newMockRepo 构造绑定 sqlmock 连接的 repository，避免依赖真实 MySQL。
// 白盒测试：直接注入 Repo{db}，验证 WithTx/tenant/CAS 的 SQL 语义。
// SkipDefaultTransaction=true：单条写不再自动包事务，便于精确断言；
// 事务语义由 TestWithTx_* 用显式 Begin/Commit/Rollback 覆盖。
func newMockRepo(t *testing.T) (*OrderRepo, sqlmock.Sqlmock) {
	t.Helper()
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New failed: %v", err)
	}
	t.Cleanup(func() { sqlDB.Close() })

	gormDB, err := gorm.Open(mysql.New(mysql.Config{
		Conn:                      sqlDB,
		SkipInitializeWithVersion: true,
	}), &gorm.Config{SkipDefaultTransaction: true})
	if err != nil {
		t.Fatalf("gorm.Open failed: %v", err)
	}
	return &OrderRepo{Repo: Repo{db: gormDB}}, mock
}

// GORM 实际生成的写 SQL（观察自首次运行）：
//   UPDATE `biz_order_payment` SET `status`=?,`updated_at`=?
//     WHERE company_id = ? AND (id = ? AND status = ?) AND `biz_order_payment`.`deleted` = ?
// 其中 updated_at 由 GORM 自动维护（动态时间用 AnyArg），deleted=0 由软删除插件注入。
const casPaymentSQL = "UPDATE `biz_order_payment` SET `status`=?,`updated_at`=? WHERE company_id = ? AND (id = ? AND status = ?) AND `biz_order_payment`.`deleted` = ?"

const casRefundSQL = "UPDATE `biz_order_refund` SET `status`=?,`updated_at`=? WHERE company_id = ? AND (id = ? AND status = ?) AND `biz_order_refund`.`deleted` = ?"

// TestConfirmPaymentPending_Success CAS 命中：仅当收款记录仍为“待核验”时更新成功。
func TestConfirmPaymentPending_Success(t *testing.T) {
	repo, mock := newMockRepo(t)

	mock.ExpectExec(regexp.QuoteMeta(casPaymentSQL)).
		WithArgs(enum.PaymentStatusConfirmed, sqlmock.AnyArg(), 1001, 55, enum.PaymentStatusPending, 0).
		WillReturnResult(sqlmock.NewResult(0, 1))

	ok, err := repo.ConfirmPaymentPending(1001, 55, map[string]interface{}{
		"status": enum.PaymentStatusConfirmed,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("expected ok=true when rows affected = 1")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sql expectations: %v", err)
	}
}

// TestConfirmPaymentPending_NoMatch CAS 未命中：记录已被并发处理（状态不再是待核验），返回 false。
func TestConfirmPaymentPending_NoMatch(t *testing.T) {
	repo, mock := newMockRepo(t)

	mock.ExpectExec(regexp.QuoteMeta(casPaymentSQL)).
		WithArgs(enum.PaymentStatusConfirmed, sqlmock.AnyArg(), 1001, 55, enum.PaymentStatusPending, 0).
		WillReturnResult(sqlmock.NewResult(0, 0))

	ok, err := repo.ConfirmPaymentPending(1001, 55, map[string]interface{}{
		"status": enum.PaymentStatusConfirmed,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("expected ok=false when rows affected = 0")
	}
}

// TestAuditRefundApplying CAS 语义同收款确认：退款单必须仍处于“申请中”才能审核。
func TestAuditRefundApplying(t *testing.T) {
	repo, mock := newMockRepo(t)

	mock.ExpectExec(regexp.QuoteMeta(casRefundSQL)).
		WithArgs(enum.RefundStatusApproved, sqlmock.AnyArg(), 2002, 88, enum.RefundStatusApplying, 0).
		WillReturnResult(sqlmock.NewResult(0, 1))

	ok, err := repo.AuditRefundApplying(2002, 88, map[string]interface{}{
		"status": enum.RefundStatusApproved,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("expected ok=true when rows affected = 1")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sql expectations: %v", err)
	}
}

// TestWithTx_Rollback 验证 WithTx 返回的副本把写操作路由到同一事务连接，
// 且事务回滚后写操作不落库（sqlmock 层面：未预期的 COMMIT 会报错）。
func TestWithTx_Rollback(t *testing.T) {
	repo, mock := newMockRepo(t)
	companyID, paymentID := int64(1001), int64(55)

	mock.ExpectBegin()
	tx := repo.conn().Begin() // 开启真实事务
	if tx.Error != nil {
		t.Fatalf("begin failed: %v", tx.Error)
	}

	mock.ExpectExec(regexp.QuoteMeta(casPaymentSQL)).
		WithArgs(enum.PaymentStatusConfirmed, sqlmock.AnyArg(), companyID, paymentID, enum.PaymentStatusPending, 0).
		WillReturnResult(sqlmock.NewResult(0, 1))

	repoTx := repo.WithTx(tx)
	ok, err := repoTx.ConfirmPaymentPending(companyID, paymentID, map[string]interface{}{
		"status": enum.PaymentStatusConfirmed,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("expected ok=true inside tx")
	}

	mock.ExpectRollback()
	if err := tx.Rollback().Error; err != nil {
		t.Fatalf("rollback failed: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sql expectations: %v", err)
	}
}

// TestWithTx_Commit 验证 WithTx 副本的写操作随事务一起提交（同一连接）。
func TestWithTx_Commit(t *testing.T) {
	repo, mock := newMockRepo(t)
	companyID, refundID := int64(2002), int64(88)

	mock.ExpectBegin()
	tx := repo.conn().Begin()
	if tx.Error != nil {
		t.Fatalf("begin failed: %v", tx.Error)
	}

	mock.ExpectExec(regexp.QuoteMeta(casRefundSQL)).
		WithArgs(enum.RefundStatusRejected, sqlmock.AnyArg(), companyID, refundID, enum.RefundStatusApplying, 0).
		WillReturnResult(sqlmock.NewResult(0, 1))

	repoTx := repo.WithTx(tx)
	ok, err := repoTx.AuditRefundApplying(companyID, refundID, map[string]interface{}{
		"status": enum.RefundStatusRejected,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("expected ok=true inside tx")
	}

	mock.ExpectCommit()
	if err := tx.Commit().Error; err != nil {
		t.Fatalf("commit failed: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sql expectations: %v", err)
	}
}

// TestGetByID_TenantFiltered 验证按主键查询自带 company_id 过滤与软删除条件（多租户隔离核心约束）。
func TestGetByID_TenantFiltered(t *testing.T) {
	repo, mock := newMockRepo(t)

	selectSQL := "SELECT * FROM `biz_order` WHERE company_id = ? AND `biz_order`.`id` = ? AND `biz_order`.`deleted` = ? ORDER BY `biz_order`.`id` LIMIT ?"
	mock.ExpectQuery(regexp.QuoteMeta(selectSQL)).
		WithArgs(1001, 42, 0, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "company_id", "status"}).
			AddRow(42, 1001, enum.OrderStatusPendingShoot))

	o, err := repo.GetByID(1001, 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if o == nil || o.ID != 42 || o.CompanyID != 1001 {
		t.Errorf("GetByID returned unexpected order: %+v", o)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sql expectations: %v", err)
	}
}
