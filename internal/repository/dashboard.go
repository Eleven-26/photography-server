package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"photography-server/internal/enum"
	"photography-server/internal/model"
)

type DashboardRepo struct {
	Repo
}

// WithTx 返回绑定到指定事务连接的副本，事务内的所有写操作将复用该连接，
// 保证跨多张表的写入原子性（失败自动回滚）。
func (r *DashboardRepo) WithTx(tx *gorm.DB) *DashboardRepo {
	return &DashboardRepo{Repo: Repo{db: tx}}
}

func NewDashboardRepo() *DashboardRepo { return &DashboardRepo{} }

// Overview 工作台聚合。字段口径与 PC 前端 DashboardOverview 对齐（原字段保留不删，
// 供其他面板复用），避免"后端自持结构直接当 API 契约"导致的字段漂移。
type Overview struct {
	// —— 工作台四卡片口径 ——
	TodayOrders       int64   `json:"today_orders"`       // 今日新增订单数
	TodayAmount       float64 `json:"today_amount"`       // 今日确认到账金额
	MonthOrders       int64   `json:"month_orders"`       // 本月新增订单数
	MonthAmount       float64 `json:"month_amount"`       // 本月确认到账金额
	PendingPayments   int64   `json:"pending_payments"`   // 待核验收款单数
	PendingDeliveries int64   `json:"pending_deliveries"` // 待交付订单数
	NewLeads          int64   `json:"new_leads"`          // 未成交未流失的线索数
	OverdueLeads      int64   `json:"overdue_leads"`      // 逾期未跟进的线索数

	// —— 扩展口径（保留）——
	PendingDeposit int64   `json:"pending_deposit"` // 待定金订单数
	PendingRetouch int64   `json:"pending_retouch"` // 精修中订单数
	UpcomingShoots int64   `json:"upcoming_shoots"` // 未来待拍摄订单数
	TodayConfirmed float64 `json:"today_confirmed"` // 今日确认到账金额（同 today_amount）
	TodayPending   float64 `json:"today_pending"`   // 今日申报待核验金额
	UnreadNotify   int64   `json:"unread_notify"`   // 当前登录人未读通知数
}

// GetOverview 聚合多条统计 SQL。ctx 透传后所有查询均挂到当前请求链路。
// 任一条统计失败即返回错误，避免"返回半份数据"让前端误判为零。
func (r *DashboardRepo) GetOverview(ctx context.Context, companyID, userID int64) (*Overview, error) {
	var ov Overview

	now := time.Now()
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
	dayEnd := dayStart.AddDate(0, 0, 1)
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)
	monthEnd := monthStart.AddDate(0, 1, 0)

	// Order.created_at 为 datetime，用时间区间比较；OrderPayment.paid_at 为字符串列，
	// 用同格式字符串比较（避免隐式转换导致索引失效）。
	dayStartStr := dayStart.Format("2006-01-02 15:04:05")
	dayEndStr := dayEnd.Format("2006-01-02 15:04:05")
	monthStartStr := monthStart.Format("2006-01-02 15:04:05")
	monthEndStr := monthEnd.Format("2006-01-02 15:04:05")
	nowStr := now.Format("2006-01-02 15:04:05")

	openLeadStatus := []int{int(enum.LeadStatusPending), int(enum.LeadStatusQuoting), int(enum.LeadStatusQuoted)}

	q := func() *gorm.DB { return r.tenant(companyID).WithContext(ctx) }

	// 今日 / 本月新增订单数
	if err := q().Model(&model.Order{}).
		Where("created_at >= ? AND created_at < ?", dayStart, dayEnd).Count(&ov.TodayOrders).Error; err != nil {
		return nil, err
	}
	if err := q().Model(&model.Order{}).
		Where("created_at >= ? AND created_at < ?", monthStart, monthEnd).Count(&ov.MonthOrders).Error; err != nil {
		return nil, err
	}

	// 今日 / 本月确认到账金额（仅统计已核验通过的收款单）
	if err := q().Model(&model.OrderPayment{}).
		Where("status = ? AND paid_at >= ? AND paid_at < ?", int(enum.PaymentStatusConfirmed), dayStartStr, dayEndStr).
		Select("COALESCE(SUM(amount),0)").Scan(&ov.TodayAmount).Error; err != nil {
		return nil, err
	}
	if err := q().Model(&model.OrderPayment{}).
		Where("status = ? AND paid_at >= ? AND paid_at < ?", int(enum.PaymentStatusConfirmed), monthStartStr, monthEndStr).
		Select("COALESCE(SUM(amount),0)").Scan(&ov.MonthAmount).Error; err != nil {
		return nil, err
	}
	ov.TodayConfirmed = ov.TodayAmount

	// 今日申报待核验金额
	if err := q().Model(&model.OrderPayment{}).
		Where("status = ? AND created_at >= ? AND created_at < ?", int(enum.PaymentStatusPending), dayStart, dayEnd).
		Select("COALESCE(SUM(amount),0)").Scan(&ov.TodayPending).Error; err != nil {
		return nil, err
	}

	// 待核验收款单数 / 待交付订单数
	if err := q().Model(&model.OrderPayment{}).
		Where("status = ?", int(enum.PaymentStatusPending)).Count(&ov.PendingPayments).Error; err != nil {
		return nil, err
	}
	if err := q().Model(&model.Order{}).
		Where("status = ?", int(enum.OrderStatusPendingDelivery)).Count(&ov.PendingDeliveries).Error; err != nil {
		return nil, err
	}

	// 待定金 / 精修中订单数
	if err := q().Model(&model.Order{}).
		Where("status = ?", int(enum.OrderStatusPendingDeposit)).Count(&ov.PendingDeposit).Error; err != nil {
		return nil, err
	}
	if err := q().Model(&model.Order{}).
		Where("status = ?", int(enum.OrderStatusRetouching)).Count(&ov.PendingRetouch).Error; err != nil {
		return nil, err
	}

	// 未来待拍摄订单：已排期（待拍摄/拍摄中）且拍摄日期不早于今天
	if err := q().Model(&model.Order{}).
		Where("status IN ? AND shoot_date IS NOT NULL AND shoot_date >= ?",
			[]int{int(enum.OrderStatusPendingShoot), int(enum.OrderStatusShooting)}, dayStart.Format("2006-01-02")).
		Count(&ov.UpcomingShoots).Error; err != nil {
		return nil, err
	}

	// 线索：跟进中 / 逾期未跟进
	if err := q().Model(&model.Lead{}).
		Where("status IN ?", openLeadStatus).Count(&ov.NewLeads).Error; err != nil {
		return nil, err
	}
	if err := q().Model(&model.Lead{}).
		Where("status IN ? AND next_follow_at IS NOT NULL AND next_follow_at <> '' AND next_follow_at < ?", openLeadStatus, nowStr).
		Count(&ov.OverdueLeads).Error; err != nil {
		return nil, err
	}

	// 当前登录人未读通知
	if err := q().Model(&model.SysNotification{}).
		Where("receiver_id = ? AND is_read = ?", userID, int(enum.NotificationUnread)).Count(&ov.UnreadNotify).Error; err != nil {
		return nil, err
	}

	return &ov, nil
}

func (r *DashboardRepo) GetCalendarBlocks(ctx context.Context, companyID int64, weekStart, weekEnd string) ([]model.CalendarBlock, error) {
	var list []model.CalendarBlock
	err := r.tenant(companyID).WithContext(ctx).Where("date BETWEEN ? AND ? AND status = ?", weekStart, weekEnd, int(enum.BlockStatusLocked)).Find(&list).Error
	return list, err
}
