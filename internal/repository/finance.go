package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"photography-server/internal/enum"
	"photography-server/internal/model"
)

type FinanceRepo struct {
	Repo
}

// WithTx 返回绑定到指定事务连接的副本，事务内的所有写操作将复用该连接，
// 保证跨多张表的写入原子性（失败自动回滚）。
func (r *FinanceRepo) WithTx(tx *gorm.DB) *FinanceRepo {
	return &FinanceRepo{Repo: Repo{db: tx}}
}

func NewFinanceRepo() *FinanceRepo { return &FinanceRepo{} }

type Summary struct {
	TotalIncome  float64 `json:"total_income"`
	DepositTotal float64 `json:"deposit_total"`
	FinalTotal   float64 `json:"final_total"`
	RefundTotal  float64 `json:"refund_total"`
	PendingCount int64   `json:"pending_count"`
}

func (r *FinanceRepo) GetSummary(ctx context.Context, companyID int64, start, end string) (*Summary, error) {
	var s Summary
	q := r.tenant(companyID).WithContext(ctx)

	q.Model(&model.OrderPayment{}).
		Where("status = ? AND paid_at BETWEEN ? AND ?", int(enum.PaymentStatusConfirmed), start, end).
		Select("COALESCE(SUM(amount),0)").Scan(&s.TotalIncome)

	q.Model(&model.OrderPayment{}).
		Where("status = ? AND type = ? AND paid_at BETWEEN ? AND ?", int(enum.PaymentStatusConfirmed), "deposit", start, end).
		Select("COALESCE(SUM(amount),0)").Scan(&s.DepositTotal)

	q.Model(&model.OrderPayment{}).
		Where("status = ? AND type IN (?) AND paid_at BETWEEN ? AND ?", int(enum.PaymentStatusConfirmed), []string{"final", "addon"}, start, end).
		Select("COALESCE(SUM(amount),0)").Scan(&s.FinalTotal)

	q.Model(&model.OrderRefund{}).
		Where("status = ? AND refund_at BETWEEN ? AND ?", int(enum.RefundStatusDone), start, end).
		Select("COALESCE(SUM(amount),0)").Scan(&s.RefundTotal)

	return &s, nil
}

func (r *FinanceRepo) ListPayments(ctx context.Context, companyID int64, page, pageSize int, status string) ([]model.OrderPayment, int64, error) {
	q := r.tenant(companyID).WithContext(ctx)
	if status != "" {
		q = q.Where("status = ?", status)
	} else {
		q = q.Where("status = ?", int(enum.PaymentStatusPending))
	}
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

func (r *FinanceRepo) ListRefunds(ctx context.Context, companyID int64, page, pageSize int) ([]model.OrderRefund, int64, error) {
	q := r.tenant(companyID).WithContext(ctx).Where("status = ?", int(enum.RefundStatusDone))
	var total int64
	if err := q.Model(&model.OrderRefund{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.OrderRefund
	page, pageSize = normalizePage(page, pageSize)
	if err := q.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func (r *FinanceRepo) GetMonthlyStats(ctx context.Context, companyID int64, year int) ([]MonthlyStat, error) {
	type row struct {
		Month  int     `json:"month"`
		Income float64 `json:"income"`
		Refund float64 `json:"refund"`
	}
	var rows []row
	start := time.Date(year, 1, 1, 0, 0, 0, 0, time.Local).Format("2006-01-02")
	end := time.Date(year, 12, 31, 23, 59, 59, 0, time.Local).Format("2006-01-02 15:04:05")
	r.tenant(companyID).WithContext(ctx).Model(&model.OrderPayment{}).
		Select("MONTH(paid_at) as month, COALESCE(SUM(amount),0) as income").
		Where("status = ? AND paid_at BETWEEN ? AND ?", int(enum.PaymentStatusConfirmed), start, end).
		Group("MONTH(paid_at)").Scan(&rows)

	var result []MonthlyStat
	for _, r := range rows {
		result = append(result, MonthlyStat{Month: r.Month, Income: r.Income, Refund: 0})
	}
	return result, nil
}

type MonthlyStat struct {
	Month  int     `json:"month"`
	Income float64 `json:"income"`
	Refund float64 `json:"refund"`
}
