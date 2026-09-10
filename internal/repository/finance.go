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

// Summary 财务汇总。字段口径与 PC 前端 FinanceSummary 对齐（原字段保留不删）。
//
// 四个金额口径的关系（原型强调「待核验申报不计入已收，但计入剩余应收」）：
//
//	MonthReceivable   本月应收 = 本月成交订单总额（SUM(orders.total_amt)）
//	MonthReceived     本月已收 = 本月已核验通过的收款金额
//	MonthRemaining    剩余应收 = 应收 - 已收（负数归零）
//	PendingVerifyAmount 待核验金额（不计入已收，但已包含在应收内）
type Summary struct {
	// —— 原型/前端口径 ——
	MonthReceivable     float64 `json:"month_receivable"`      // 本月应收（订单总额）
	MonthReceived       float64 `json:"month_received"`        // 本月已确认到账
	MonthRemaining      float64 `json:"month_remaining"`       // 本月剩余应收
	PendingVerifyCount  int64   `json:"pending_verify_count"`  // 待核验申报笔数
	PendingVerifyAmount float64 `json:"pending_verify_amount"` // 待核验申报金额
	RefundingCount      int64   `json:"refunding_count"`       // 退款中笔数
	RefundingAmount     float64 `json:"refunding_amount"`      // 退款中金额

	// —— 扩展口径（保留）——
	TotalIncome  float64 `json:"total_income"`  // 区间内已确认到账（同 MonthReceived）
	DepositTotal float64 `json:"deposit_total"` // 区间内定金到账
	FinalTotal   float64 `json:"final_total"`   // 区间内尾款+加片到账
	RefundTotal  float64 `json:"refund_total"`  // 区间内已退款金额
	PendingCount int64   `json:"pending_count"` // 待核验笔数（同 PendingVerifyCount）
}

// GetSummary 财务汇总。start/end 为闭区间时间字符串（"2006-01-02 15:04:05"）。
// 任一条统计失败即返回错误，避免半份数据被当成真实财务口径。
func (r *FinanceRepo) GetSummary(ctx context.Context, companyID int64, start, end string) (*Summary, error) {
	var s Summary
	q := func() *gorm.DB { return r.tenant(companyID).WithContext(ctx) }

	// 本月应收：本月创建的订单总额
	if err := q().Model(&model.Order{}).
		Where("created_at >= ? AND created_at < ?", start, end).
		Select("COALESCE(SUM(total_amt),0)").Scan(&s.MonthReceivable).Error; err != nil {
		return nil, err
	}

	// 本月已收 / 定金 / 尾款+加片
	if err := q().Model(&model.OrderPayment{}).
		Where("status = ? AND paid_at >= ? AND paid_at < ?", int(enum.PaymentStatusConfirmed), start, end).
		Select("COALESCE(SUM(amount),0)").Scan(&s.MonthReceived).Error; err != nil {
		return nil, err
	}
	if err := q().Model(&model.OrderPayment{}).
		Where("status = ? AND type = ? AND paid_at >= ? AND paid_at < ?", int(enum.PaymentStatusConfirmed), "deposit", start, end).
		Select("COALESCE(SUM(amount),0)").Scan(&s.DepositTotal).Error; err != nil {
		return nil, err
	}
	if err := q().Model(&model.OrderPayment{}).
		Where("status = ? AND type IN (?) AND paid_at >= ? AND paid_at < ?", int(enum.PaymentStatusConfirmed), []string{"final", "addon"}, start, end).
		Select("COALESCE(SUM(amount),0)").Scan(&s.FinalTotal).Error; err != nil {
		return nil, err
	}
	s.TotalIncome = s.MonthReceived
	if s.MonthRemaining = s.MonthReceivable - s.MonthReceived; s.MonthRemaining < 0 {
		s.MonthRemaining = 0
	}

	// 待核验申报：不受月份限制（属于待办口径，本月口径会漏掉历史欠核验单）
	if err := q().Model(&model.OrderPayment{}).
		Where("status = ?", int(enum.PaymentStatusPending)).
		Count(&s.PendingVerifyCount).Error; err != nil {
		return nil, err
	}
	if err := q().Model(&model.OrderPayment{}).
		Where("status = ?", int(enum.PaymentStatusPending)).
		Select("COALESCE(SUM(amount),0)").Scan(&s.PendingVerifyAmount).Error; err != nil {
		return nil, err
	}
	s.PendingCount = s.PendingVerifyCount

	// 退款中（申请中 + 已通过未打款）
	if err := q().Model(&model.OrderRefund{}).
		Where("status IN ?", []int{int(enum.RefundStatusApplying), int(enum.RefundStatusApproved)}).
		Count(&s.RefundingCount).Error; err != nil {
		return nil, err
	}
	if err := q().Model(&model.OrderRefund{}).
		Where("status IN ?", []int{int(enum.RefundStatusApplying), int(enum.RefundStatusApproved)}).
		Select("COALESCE(SUM(amount),0)").Scan(&s.RefundingAmount).Error; err != nil {
		return nil, err
	}

	// 已退款金额（区间内）
	if err := q().Model(&model.OrderRefund{}).
		Where("status = ? AND refund_at >= ? AND refund_at < ?", int(enum.RefundStatusDone), start, end).
		Select("COALESCE(SUM(amount),0)").Scan(&s.RefundTotal).Error; err != nil {
		return nil, err
	}

	return &s, nil
}

// ListPayments 收款流水。status 为空表示不筛选（全部状态）。
// 旧实现空值默认只查"待核验"，导致前端下拉的"全部状态"名不副实。
func (r *FinanceRepo) ListPayments(ctx context.Context, companyID int64, page, pageSize int, status string) ([]model.OrderPayment, int64, error) {
	q := r.tenant(companyID).WithContext(ctx)
	if status != "" {
		q = q.Where("status = ?", status)
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

// ListRefunds 退款流水。status 为空表示不筛选（全部状态）。
// 旧实现硬编码 status=已退款，导致"退款审批"页看不到申请中/已通过的单据。
func (r *FinanceRepo) ListRefunds(ctx context.Context, companyID int64, page, pageSize int, status string) ([]model.OrderRefund, int64, error) {
	q := r.tenant(companyID).WithContext(ctx)
	if status != "" {
		q = q.Where("status = ?", status)
	}
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
