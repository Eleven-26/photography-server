package repository

import (
	"gorm.io/gorm"

	"photography-server/internal/model"
)

type OrderRepo struct{}

func NewOrderRepo() *OrderRepo {
	return &OrderRepo{}
}

// List 订单列表（分页 + 关键字 + 状态 + 客户 + 摄影师 + 日期筛选）
func (r *OrderRepo) List(companyID int64, page, pageSize int, keyword, status string, customerID, photographerID int64, date string) ([]model.Order, int64, error) {
	q := tenant(companyID)
	if keyword != "" {
		kw := "%" + keyword + "%"
		q = q.Where("code LIKE ? OR customer_name LIKE ? OR customer_mobile LIKE ?", kw, kw, kw)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if customerID > 0 {
		q = q.Where("customer_id = ?", customerID)
	}
	if photographerID > 0 {
		q = q.Where("photographer_id = ?", photographerID)
	}
	if date != "" {
		q = q.Where("shoot_date = ?", date)
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

// GetByID 根据 ID 查询订单
func (r *OrderRepo) GetByID(companyID, id int64) (*model.Order, error) {
	var o model.Order
	if err := tenant(companyID).First(&o, id).Error; err != nil {
		return nil, err
	}
	return &o, nil
}

// Create 创建订单
func (r *OrderRepo) Create(o *model.Order) error {
	return tenant(o.CompanyID).Create(o).Error
}

// Update 更新订单
func (r *OrderRepo) Update(companyID, id int64, updates map[string]interface{}) error {
	return tenant(companyID).Model(&model.Order{}).Where("id = ?", id).Updates(updates).Error
}

// ListPayments 查询订单收款记录
func (r *OrderRepo) ListPayments(companyID, orderID int64) ([]model.OrderPayment, error) {
	var list []model.OrderPayment
	err := tenant(companyID).Where("order_id = ?", orderID).Order("id ASC").Find(&list).Error
	return list, err
}

// ListRefunds 查询订单退款记录
func (r *OrderRepo) ListRefunds(companyID, orderID int64) ([]model.OrderRefund, error) {
	var list []model.OrderRefund
	err := tenant(companyID).Where("order_id = ?", orderID).Order("id ASC").Find(&list).Error
	return list, err
}

// ListLogs 查询订单操作日志
func (r *OrderRepo) ListLogs(companyID, orderID int64) ([]model.OrderLog, error) {
	var list []model.OrderLog
	err := tenant(companyID).Where("order_id = ?", orderID).Order("id ASC").Find(&list).Error
	return list, err
}

// GetDelivery 查询订单交付单
func (r *OrderRepo) GetDelivery(companyID, orderID int64) (*model.Delivery, error) {
	var dv model.Delivery
	if err := tenant(companyID).Where("order_id = ?", orderID).First(&dv).Error; err != nil {
		return nil, err
	}
	return &dv, nil
}

// CreateLog 创建订单操作日志
func (r *OrderRepo) CreateLog(tx *gorm.DB, l *model.OrderLog) error {
	return tx.Create(l).Error
}

// CreateCalendarBlock 创建档期
func (r *OrderRepo) CreateCalendarBlock(tx *gorm.DB, b *model.CalendarBlock) error {
	return tx.Create(b).Error
}

// UpdateCalendarBlockByOrder 更新关联档期
func (r *OrderRepo) UpdateCalendarBlockByOrder(tx *gorm.DB, orderID int64, updates map[string]interface{}) error {
	return tx.Model(&model.CalendarBlock{}).Where("order_id = ?", orderID).Updates(updates).Error
}

// CancelCalendarBlockByOrder 取消关联档期
func (r *OrderRepo) CancelCalendarBlockByOrder(tx *gorm.DB, orderID int64) error {
	return tx.Model(&model.CalendarBlock{}).Where("order_id = ?", orderID).Update("status", model.BlockStatusCancelled).Error
}

// CreatePayment 创建收款记录
func (r *OrderRepo) CreatePayment(p *model.OrderPayment) error {
	return tenant(p.CompanyID).Create(p).Error
}

// GetPaymentByID 根据 ID 查询收款记录
func (r *OrderRepo) GetPaymentByID(companyID, id int64) (*model.OrderPayment, error) {
	var p model.OrderPayment
	if err := tenant(companyID).First(&p, id).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

// UpdatePayment 更新收款记录
func (r *OrderRepo) UpdatePayment(companyID, id int64, updates map[string]interface{}) error {
	return tenant(companyID).Model(&model.OrderPayment{}).Where("id = ?", id).Updates(updates).Error
}

// DeletePayment 删除收款记录
func (r *OrderRepo) DeletePayment(tx *gorm.DB, id int64) error {
	return tx.Delete(&model.OrderPayment{}, id).Error
}

// GetRefundByID 根据 ID 查询退款记录
func (r *OrderRepo) GetRefundByID(companyID, id int64) (*model.OrderRefund, error) {
	var rf model.OrderRefund
	if err := tenant(companyID).First(&rf, id).Error; err != nil {
		return nil, err
	}
	return &rf, nil
}

// CreateRefund 创建退款记录
func (r *OrderRepo) CreateRefund(rf *model.OrderRefund) error {
	return tenant(rf.CompanyID).Create(rf).Error
}

// UpdateRefund 更新退款记录
func (r *OrderRepo) UpdateRefund(companyID, id int64, updates map[string]interface{}) error {
	return tenant(companyID).Model(&model.OrderRefund{}).Where("id = ?", id).Updates(updates).Error
}
