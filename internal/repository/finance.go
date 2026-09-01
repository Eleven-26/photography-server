package repository

import (
	"photography-server/internal/model"
)

type FinanceRepo struct{}

func NewFinanceRepo() *FinanceRepo {
	return &FinanceRepo{}
}

// Summary 财务汇总
func (r *FinanceRepo) Summary(companyID int64, start, end string) (*FinanceSummary, error) {
	sum := &FinanceSummary{}
	q := tenant(companyID)
	if err := q.Model(&model.OrderPayment{}).
		Where("status = ? AND paid_at BETWEEN ? AND ?", model.PaymentStatusConfirmed, start, end).
		Select("COALESCE(SUM(amount),0)").Scan(&sum.Revenue).Error; err != nil {
		return nil, err
	}
	q.Model(&model.OrderPayment{}).
		Where("status = ? AND type = ? AND paid_at BETWEEN ? AND ?", model.PaymentStatusConfirmed, "deposit", start, end).
		Select("COALESCE(SUM(amount),0)").Scan(&sum.DepositIncome)
	q.Model(&model.OrderPayment{}).
		Where("status = ? AND type IN (?) AND paid_at BETWEEN ? AND ?", model.PaymentStatusConfirmed, []string{"final", "addon"}, start, end).
		Select("COALESCE(SUM(amount),0)").Scan(&sum.FinalIncome)
	q.Model(&model.OrderRefund{}).
		Where("status = ? AND refund_at BETWEEN ? AND ?", model.RefundStatusDone, start, end).
		Select("COALESCE(SUM(amount),0)").Scan(&sum.RefundTotal)
	q.Model(&model.Order{}).
		Where("created_at BETWEEN ? AND ?", start, end).
		Count(&sum.OrderCount)
	q.Model(&model.OrderPayment{}).
		Where("status = ?", model.PaymentStatusPending).
		Count(&sum.PendingCount)
	return sum, nil
}

// ListPayments 已确认收款列表（分页 + 日期范围）
func (r *FinanceRepo) ListPayments(companyID int64, page, pageSize int, startDate, endDate string) ([]model.OrderPayment, int64, error) {
	q := tenant(companyID).Where("status = ?", model.PaymentStatusConfirmed)
	if startDate != "" {
		q = q.Where("paid_at >= ?", startDate+" 00:00:00")
	}
	if endDate != "" {
		q = q.Where("paid_at <= ?", endDate+" 23:59:59")
	}
	var total int64
	if err := q.Model(&model.OrderPayment{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.OrderPayment
	page, pageSize = normalizePage(page, pageSize)
	if err := q.Order("paid_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// ListRefunds 已退款列表（分页 + 日期范围）
func (r *FinanceRepo) ListRefunds(companyID int64, page, pageSize int, startDate, endDate string) ([]model.OrderRefund, int64, error) {
	q := tenant(companyID).Where("status = ?", model.RefundStatusDone)
	if startDate != "" {
		q = q.Where("refund_at >= ?", startDate+" 00:00:00")
	}
	if endDate != "" {
		q = q.Where("refund_at <= ?", endDate+" 23:59:59")
	}
	var total int64
	if err := q.Model(&model.OrderRefund{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.OrderRefund
	page, pageSize = normalizePage(page, pageSize)
	if err := q.Order("refund_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// FinanceSummary 财务汇总响应
type FinanceSummary struct {
	Month         string  `json:"month"`
	Revenue       float64 `json:"revenue"`
	DepositIncome float64 `json:"deposit_income"`
	FinalIncome   float64 `json:"final_income"`
	RefundTotal   float64 `json:"refund_total"`
	OrderCount    int64   `json:"order_count"`
	PendingCount  int64   `json:"pending_count"`
}
