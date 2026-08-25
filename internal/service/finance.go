package service

import (
	"time"

	"photography-server/internal/model"
)

type FinanceSummary struct {
	Month         string  `json:"month"`
	Revenue       float64 `json:"revenue"`        // 本月实收（已确认收款）
	DepositIncome float64 `json:"deposit_income"` // 本月定金收入
	FinalIncome   float64 `json:"final_income"`   // 本月尾款/加选收入
	RefundTotal   float64 `json:"refund_total"`   // 本月退款
	OrderCount    int64   `json:"order_count"`    // 本月新订单数
	PendingCount  int64   `json:"pending_count"`  // 待核验收款数
}

func monthRange(month string) (string, string) {
	if len(month) != 7 {
		month = time.Now().Format("2006-01")
	}
	return month + "-01 00:00:00", month + "-31 23:59:59"
}

func (s *Service) FinanceSummary(op Operator, month string) (*FinanceSummary, error) {
	start, end := monthRange(month)
	sum := &FinanceSummary{Month: month}

	q := s.tenant(op)
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

func (s *Service) ListFinancePayments(op Operator, page, pageSize int, startDate, endDate string) ([]model.OrderPayment, int64, error) {
	q := s.tenant(op).Where("status = ?", model.PaymentStatusConfirmed)
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

func (s *Service) ListFinanceRefunds(op Operator, page, pageSize int, startDate, endDate string) ([]model.OrderRefund, int64, error) {
	q := s.tenant(op).Where("status = ?", model.RefundStatusDone)
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
