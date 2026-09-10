package repository

import (
	"context"
	"math"
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

	// —— P2 新增聚合口径（原型工作台）——
	MonthLeads     int64        `json:"month_leads"`     // 本月新增线索数
	MonthDealRate  float64      `json:"month_deal_rate"` // 本月成交率 %（本月新增订单 / 本月新增线索）
	AvailableSlots int64        `json:"available_slots"` // 未来 7 天剩余可约时段数
	TodayShoots    []TodayShoot `json:"today_shoots"`    // 今日拍摄列表
	TodoItems      []TodoItem   `json:"todo_items"`      // 待办清单
}

// TodayShoot 今日拍摄条目（工作台「今日拍摄」列表）
type TodayShoot struct {
	ID           int64  `json:"id"`
	Code         string `json:"code"`
	CustomerName string `json:"customer_name"`
	PackageName  string `json:"package_name"`
	ShootTime    string `json:"shoot_time"`
	ShootAddress string `json:"shoot_address"`
	Photographer string `json:"photographer"`
	Status       int    `json:"status"`
}

// TodoItem 待办条目（工作台「待办清单」）。Count 为 0 的条目在 service 层过滤掉，
// 前端只渲染真正需要处理的事项。
type TodoItem struct {
	Key   string `json:"key"`   // 唯一标识，前端用作 v-for key
	Label string `json:"label"` // 展示文案
	Count int64  `json:"count"` // 待处理数量
	Route string `json:"route"` // 点击跳转的前端路由
	Tone  string `json:"tone"`  // 视觉色调：warning / danger / normal
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

	// 本月新增线索：作为成交率分母（与「本月新增订单」取同一时间窗，口径可解释）
	if err := q().Model(&model.Lead{}).
		Where("created_at >= ? AND created_at < ?", monthStart, monthEnd).Count(&ov.MonthLeads).Error; err != nil {
		return nil, err
	}
	if ov.MonthLeads > 0 {
		ov.MonthDealRate = math.Round(float64(ov.MonthOrders)/float64(ov.MonthLeads)*1000) / 10
	}

	// 今日拍摄列表（待拍摄 / 拍摄中，按时间排序）
	if err := q().Model(&model.Order{}).
		Select("id, code, customer_name, package_name, shoot_time, shoot_address, photographer, status").
		Where("shoot_date = ? AND status IN ?", dayStart.Format("2006-01-02"),
			[]int{int(enum.OrderStatusPendingShoot), int(enum.OrderStatusShooting)}).
		Order("shoot_time ASC").Scan(&ov.TodayShoots).Error; err != nil {
		return nil, err
	}

	// 未来 7 天剩余可约时段
	slots, err := r.availableSlots(ctx, companyID, dayStart, 7)
	if err != nil {
		return nil, err
	}
	ov.AvailableSlots = slots

	// 待办清单：复用上面的计数，只保留 count > 0 的条目
	ov.TodoItems = buildTodoItems(&ov)

	return &ov, nil
}

// availableSlots 计算自 start 起 days 天内的剩余可约时段数。
// 口径：某天存在启用中的档期模板即计 1 个可约时段，减去该天已锁定（未取消）的档期块，
// 负数归零；多条模板（不同摄影师/时段）按条累加。
func (r *DashboardRepo) availableSlots(ctx context.Context, companyID int64, start time.Time, days int) (int64, error) {
	q := func() *gorm.DB { return r.tenant(companyID).WithContext(ctx) }

	var templates []model.SlotTemplate
	if err := q().Model(&model.SlotTemplate{}).Where("status = ?", 1).Find(&templates).Error; err != nil {
		return 0, err
	}
	if len(templates) == 0 {
		return 0, nil
	}

	end := start.AddDate(0, 0, days)
	type dateCount struct {
		Date string `gorm:"column:date"`
		Cnt  int64  `gorm:"column:cnt"`
	}
	var locks []dateCount
	if err := q().Model(&model.CalendarBlock{}).
		Select("date, COUNT(*) AS cnt").
		Where("status = ? AND date >= ? AND date < ?",
			int(enum.BlockStatusLocked), start.Format("2006-01-02"), end.Format("2006-01-02")).
		Group("date").Scan(&locks).Error; err != nil {
		return 0, err
	}
	lockByDate := make(map[string]int64, len(locks))
	for _, l := range locks {
		lockByDate[l.Date] = l.Cnt
	}

	var total int64
	for i := 0; i < days; i++ {
		d := start.AddDate(0, 0, i)
		wd := int(d.Weekday()) // 0=周日，与 SlotTemplate.Weekday 取值一致
		var n int64
		for _, tpl := range templates {
			if tpl.Weekday == wd {
				n++
			}
		}
		n -= lockByDate[d.Format("2006-01-02")]
		if n > 0 {
			total += n
		}
	}
	return total, nil
}

// buildTodoItems 由概览计数组装待办清单（count 为 0 的条目不返回）
func buildTodoItems(ov *Overview) []TodoItem {
	candidates := []TodoItem{
		{Key: "overdue_lead", Label: "逾期未跟进线索", Count: ov.OverdueLeads, Route: "/leads", Tone: "danger"},
		{Key: "pending_payment", Label: "待核验收款", Count: ov.PendingPayments, Route: "/finance", Tone: "warning"},
		{Key: "pending_delivery", Label: "待交付订单", Count: ov.PendingDeliveries, Route: "/delivery", Tone: "warning"},
		{Key: "pending_retouch", Label: "精修进行中", Count: ov.PendingRetouch, Route: "/delivery", Tone: "normal"},
		{Key: "pending_deposit", Label: "待收定金", Count: ov.PendingDeposit, Route: "/orders", Tone: "normal"},
	}
	out := make([]TodoItem, 0, len(candidates))
	for _, it := range candidates {
		if it.Count > 0 {
			out = append(out, it)
		}
	}
	return out
}

func (r *DashboardRepo) GetCalendarBlocks(ctx context.Context, companyID int64, weekStart, weekEnd string) ([]model.CalendarBlock, error) {
	var list []model.CalendarBlock
	err := r.tenant(companyID).WithContext(ctx).Where("date BETWEEN ? AND ? AND status = ?", weekStart, weekEnd, int(enum.BlockStatusLocked)).Find(&list).Error
	return list, err
}
