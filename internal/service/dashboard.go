package service

import (
	"time"

	"photography-server/internal/model"
)

type DashboardOverview struct {
	TodayNewOrders   int64                 `json:"today_new_orders"`  // 今日新增订单
	PendingDeposit   int64                 `json:"pending_deposit"`   // 待收定金订单
	UpcomingShoots   int64                 `json:"upcoming_shoots"`   // 近7天待拍摄
	MonthRevenue     float64               `json:"month_revenue"`     // 本月实收
	UnreadNotices    int64                 `json:"unread_notices"`    // 未读通知
	RecentOrders     []model.Order         `json:"recent_orders"`     // 最近订单
	UpcomingCalendar []model.CalendarBlock `json:"upcoming_calendar"` // 近7天档期
}

func (s *Service) Overview(op Operator) (*DashboardOverview, error) {
	ov := &DashboardOverview{}
	now := time.Now()
	todayStart := now.Format("2006-01-02") + " 00:00:00"
	todayEnd := now.Format("2006-01-02") + " 23:59:59"
	weekStart := now.Format("2006-01-02")
	weekEnd := now.AddDate(0, 0, 7).Format("2006-01-02")

	q := s.tenant(op)
	q.Model(&model.Order{}).Where("created_at BETWEEN ? AND ?", todayStart, todayEnd).Count(&ov.TodayNewOrders)
	q.Model(&model.Order{}).Where("status = ?", model.OrderStatusPendingDeposit).Count(&ov.PendingDeposit)
	q.Model(&model.Order{}).Where("status NOT IN ? AND shoot_date BETWEEN ? AND ?",
		[]string{model.OrderStatusCompleted, model.OrderStatusCancelled}, weekStart, weekEnd).Count(&ov.UpcomingShoots)
	q.Model(&model.OrderPayment{}).Where("status = ?", model.PaymentStatusConfirmed).
		Select("COALESCE(SUM(amount),0)").
		Where("paid_at BETWEEN ? AND ?", now.Format("2006-01")+"-01 00:00:00", now.Format("2006-01")+"-31 23:59:59").
		Scan(&ov.MonthRevenue)
	q.Model(&model.SysNotification{}).Where("receiver_id = ? AND is_read = ?", op.UserID, model.NotificationUnread).
		Count(&ov.UnreadNotices)
	q.Order("id DESC").Limit(10).Find(&ov.RecentOrders)
	q.Where("date BETWEEN ? AND ? AND status = ?", weekStart, weekEnd, model.BlockStatusLocked).
		Order("date ASC").Find(&ov.UpcomingCalendar)
	return ov, nil
}
