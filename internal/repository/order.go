package repository

import (
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"photography-server/internal/enum"
	"photography-server/internal/model"
)

type OrderRepo struct {
	Repo
}

// WithTx 返回绑定到指定事务连接的副本，事务内的所有写操作将复用该连接，
// 保证跨多张表的写入原子性（失败自动回滚）。
func (r *OrderRepo) WithTx(tx *gorm.DB) *OrderRepo {
	return &OrderRepo{Repo: Repo{db: tx}}
}

func NewOrderRepo() *OrderRepo { return &OrderRepo{} }

func (r *OrderRepo) List(companyID int64, page, pageSize int, status string, customerID int64) ([]model.Order, int64, error) {
	q := r.tenant(companyID)
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if customerID > 0 {
		q = q.Where("customer_id = ?", customerID)
	}
	var total int64
	if err := q.Model(&model.Order{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.Order
	page, pageSize = normalizePage(page, pageSize)
	if err := q.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func (r *OrderRepo) GetByID(companyID, orderID int64) (*model.Order, error) {
	var o model.Order
	if err := r.tenant(companyID).First(&o, orderID).Error; err != nil {
		return nil, err
	}
	return &o, nil
}

// GetByIDForUpdate 事务内按主键加行锁读取订单（SELECT ... FOR UPDATE），
// 用于资金操作（确认收款/审核退款），避免并发场景下重复累加。
func (r *OrderRepo) GetByIDForUpdate(companyID, orderID int64) (*model.Order, error) {
	var o model.Order
	if err := r.tenant(companyID).Clauses(clause.Locking{Strength: "UPDATE"}).First(&o, orderID).Error; err != nil {
		return nil, err
	}
	return &o, nil
}

func (r *OrderRepo) Update(companyID, orderID int64, updates map[string]interface{}) error {
	return r.tenant(companyID).Model(&model.Order{}).Where("id = ?", orderID).Updates(updates).Error
}

// Create 创建订单主记录（company_id 由调用方在 model 上填充；
// 事务内请使用 WithTx(tx).Create(...) 复用事务连接）
func (r *OrderRepo) Create(o *model.Order) error {
	return r.conn().Create(o).Error
}

func (r *OrderRepo) UpdateCalendarBlockStatus(companyID, orderID int64, status enum.BlockStatus) error {
	return r.tenant(companyID).Model(&model.CalendarBlock{}).Where("order_id = ?", orderID).Update("status", status).Error
}

func (r *OrderRepo) CreatePayment(p *model.OrderPayment) error {
	return r.conn().Create(p).Error
}

// UpdatePayment 按公司+记录ID更新收款记录（多租户过滤，供确认收款等场景使用）
func (r *OrderRepo) UpdatePayment(companyID, paymentID int64, updates map[string]interface{}) error {
	return r.tenant(companyID).Model(&model.OrderPayment{}).Where("id = ?", paymentID).Updates(updates).Error
}

// ConfirmPaymentPending 条件更新（CAS）：仅当收款记录仍为“待核验”时置为已确认。
// 返回 false 表示该记录已被并发处理（重复点击确认不会导致金额重复累加）。
func (r *OrderRepo) ConfirmPaymentPending(companyID, paymentID int64, updates map[string]interface{}) (bool, error) {
	res := r.tenant(companyID).Model(&model.OrderPayment{}).
		Where("id = ? AND status = ?", paymentID, enum.PaymentStatusPending).
		Updates(updates)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

func (r *OrderRepo) GetPaymentByID(companyID, paymentID int64) (*model.OrderPayment, error) {
	var p model.OrderPayment
	if err := r.tenant(companyID).First(&p, paymentID).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *OrderRepo) ListPayments(companyID, orderID int64) ([]model.OrderPayment, error) {
	var list []model.OrderPayment
	err := r.tenant(companyID).Where("order_id = ?", orderID).Order("id ASC").Find(&list).Error
	return list, err
}

func (r *OrderRepo) GetUnconfirmedPayments(companyID int64, page, pageSize int) ([]model.OrderPayment, int64, error) {
	q := r.tenant(companyID).Where("status = ?", enum.PaymentStatusPending)
	var total int64
	if err := q.Model(&model.OrderPayment{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.OrderPayment
	page, pageSize = normalizePage(page, pageSize)
	if err := q.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func (r *OrderRepo) GetTodayStats(companyID int64) (confirmed float64, pending float64, err error) {
	today := "2006-01-02"
	q := r.tenant(companyID).
		Where("status = ? AND paid_at BETWEEN ? AND ?", enum.PaymentStatusConfirmed, today, today+" 23:59:59")
	if err := q.Model(&model.OrderPayment{}).Select("COALESCE(SUM(amount),0)").Scan(&confirmed).Error; err != nil {
		return 0, 0, err
	}
	q2 := r.tenant(companyID).
		Where("status = ? AND created_at BETWEEN ? AND ?", enum.PaymentStatusPending, today, today+" 23:59:59")
	if err := q2.Model(&model.OrderPayment{}).Select("COALESCE(SUM(amount),0)").Scan(&pending).Error; err != nil {
		return 0, 0, err
	}
	return
}

func (r *OrderRepo) CreateRefund(refund *model.OrderRefund) error {
	return r.conn().Create(refund).Error
}

// UpdateRefund 按公司+记录ID更新退款单（多租户过滤，供审核退款等场景使用）
func (r *OrderRepo) UpdateRefund(companyID, refundID int64, updates map[string]interface{}) error {
	return r.tenant(companyID).Model(&model.OrderRefund{}).Where("id = ?", refundID).Updates(updates).Error
}

// AuditRefundApplying 条件更新（CAS）：仅当退款单仍为“申请中”时写入审核结果。
// 返回 false 表示该退款单已被并发审核，防止重复通过导致退款金额重复累加。
func (r *OrderRepo) AuditRefundApplying(companyID, refundID int64, updates map[string]interface{}) (bool, error) {
	res := r.tenant(companyID).Model(&model.OrderRefund{}).
		Where("id = ? AND status = ?", refundID, enum.RefundStatusApplying).
		Updates(updates)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

func (r *OrderRepo) GetRefundByID(companyID, refundID int64) (*model.OrderRefund, error) {
	var rf model.OrderRefund
	if err := r.tenant(companyID).First(&rf, refundID).Error; err != nil {
		return nil, err
	}
	return &rf, nil
}

func (r *OrderRepo) ListRefunds(companyID, orderID int64) ([]model.OrderRefund, error) {
	var list []model.OrderRefund
	err := r.tenant(companyID).Where("order_id = ?", orderID).Order("id ASC").Find(&list).Error
	return list, err
}

func (r *OrderRepo) CreateLog(log *model.OrderLog) error {
	return r.conn().Create(log).Error
}

func (r *OrderRepo) ListLogs(companyID, orderID int64) ([]model.OrderLog, error) {
	var list []model.OrderLog
	err := r.tenant(companyID).Where("order_id = ?", orderID).Order("id ASC").Find(&list).Error
	return list, err
}
