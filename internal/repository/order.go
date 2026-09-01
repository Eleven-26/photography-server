package repository

import (
	"photography-server/internal/enum"
	"photography-server/internal/infrastructure"
	"photography-server/internal/model"
)

type OrderRepo struct{}

func NewOrderRepo() *OrderRepo { return &OrderRepo{} }

func (r *OrderRepo) List(companyID int64, page, pageSize int, status string, customerID int64) ([]model.Order, int64, error) {
	q := tenant(companyID)
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
	if err := tenant(companyID).First(&o, orderID).Error; err != nil {
		return nil, err
	}
	return &o, nil
}

func (r *OrderRepo) Update(companyID, orderID int64, updates map[string]interface{}) error {
	return tenant(companyID).Model(&model.Order{}).Where("id = ?", orderID).Updates(updates).Error
}

func (r *OrderRepo) UpdateCalendarBlockStatus(companyID, orderID int64, status enum.BlockStatus) error {
	return tenant(companyID).Model(&model.CalendarBlock{}).Where("order_id = ?", orderID).Update("status", status).Error
}

func (r *OrderRepo) CreatePayment(p *model.OrderPayment) error {
	return infrastructure.MySQL().Create(p).Error
}

func (r *OrderRepo) GetPaymentByID(companyID, paymentID int64) (*model.OrderPayment, error) {
	var p model.OrderPayment
	if err := tenant(companyID).First(&p, paymentID).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *OrderRepo) ListPayments(companyID, orderID int64) ([]model.OrderPayment, error) {
	var list []model.OrderPayment
	err := tenant(companyID).Where("order_id = ?", orderID).Order("id ASC").Find(&list).Error
	return list, err
}

func (r *OrderRepo) GetUnconfirmedPayments(companyID int64, page, pageSize int) ([]model.OrderPayment, int64, error) {
	q := tenant(companyID).Where("status = ?", enum.PaymentStatusPending)
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
	q := tenant(companyID).
		Where("status = ? AND paid_at BETWEEN ? AND ?", enum.PaymentStatusConfirmed, today, today+" 23:59:59")
	if err := q.Model(&model.OrderPayment{}).Select("COALESCE(SUM(amount),0)").Scan(&confirmed).Error; err != nil {
		return 0, 0, err
	}
	q2 := tenant(companyID).
		Where("status = ? AND created_at BETWEEN ? AND ?", enum.PaymentStatusPending, today, today+" 23:59:59")
	if err := q2.Model(&model.OrderPayment{}).Select("COALESCE(SUM(amount),0)").Scan(&pending).Error; err != nil {
		return 0, 0, err
	}
	return
}

func (r *OrderRepo) CreateRefund(refund *model.OrderRefund) error {
	return infrastructure.MySQL().Create(refund).Error
}

func (r *OrderRepo) GetRefundByID(companyID, refundID int64) (*model.OrderRefund, error) {
	var rf model.OrderRefund
	if err := tenant(companyID).First(&rf, refundID).Error; err != nil {
		return nil, err
	}
	return &rf, nil
}

func (r *OrderRepo) ListRefunds(companyID, orderID int64) ([]model.OrderRefund, error) {
	var list []model.OrderRefund
	err := tenant(companyID).Where("order_id = ?", orderID).Order("id ASC").Find(&list).Error
	return list, err
}

func (r *OrderRepo) CreateLog(log *model.OrderLog) error {
	return infrastructure.MySQL().Create(log).Error
}

func (r *OrderRepo) ListLogs(companyID, orderID int64) ([]model.OrderLog, error) {
	var list []model.OrderLog
	err := tenant(companyID).Where("order_id = ?", orderID).Order("id ASC").Find(&list).Error
	return list, err
}
