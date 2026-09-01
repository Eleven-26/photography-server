package repository

import (
	"time"

	"photography-server/internal/enum"
	"photography-server/internal/model"
)

type DashboardRepo struct{}

func NewDashboardRepo() *DashboardRepo { return &DashboardRepo{} }

type Overview struct {
	PendingDeposit  int64   `json:"pending_deposit"`
	PendingRetouch  int64   `json:"pending_retouch"`
	UpcomingShoots  int64   `json:"upcoming_shoots"`
	PendingPayments int64   `json:"pending_payments"`
	TodayConfirmed  float64 `json:"today_confirmed"`
	TodayPending    float64 `json:"today_pending"`
	UnreadNotify    int64   `json:"unread_notify"`
}

func (r *DashboardRepo) GetOverview(companyID, userID int64) (*Overview, error) {
	var ov Overview
	q := tenant(companyID)

	q.Model(&model.Order{}).Where("status = ?", int(enum.OrderStatusPendingDeposit)).Count(&ov.PendingDeposit)
	q.Model(&model.Order{}).Where("status = ?", int(enum.OrderStatusRetouching)).Count(&ov.PendingRetouch)

	weekStart := time.Now().Format("2006-01-02")
	q.Model(&model.Order{}).Where("status IN ? AND shoot_date >= ?",
		[]int{int(enum.OrderStatusCompleted), int(enum.OrderStatusCancelled)}, weekStart).Count(&ov.UpcomingShoots)

	q.Model(&model.OrderPayment{}).Where("status = ?", int(enum.PaymentStatusPending)).Count(&ov.PendingPayments)

	q.Model(&model.SysNotification{}).Where("receiver_id = ? AND is_read = ?", userID, int(enum.NotificationUnread)).Count(&ov.UnreadNotify)

	return &ov, nil
}

func (r *DashboardRepo) GetCalendarBlocks(companyID int64, weekStart, weekEnd string) ([]model.CalendarBlock, error) {
	var list []model.CalendarBlock
	err := tenant(companyID).Where("date BETWEEN ? AND ? AND status = ?", weekStart, weekEnd, int(enum.BlockStatusLocked)).Find(&list).Error
	return list, err
}
