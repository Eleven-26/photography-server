package dto

// DashboardOverviewResp 工作台概览（PC 首页）。
//
// 与 repository 层的聚合结构显式解耦：repository 只负责取数，
// 对外契约在本包声明，避免「后端自持结构体直接当 API 契约」导致的字段静默漂移。
type DashboardOverviewResp struct {
	// —— 首页四卡片口径 ——
	TodayOrders       int64   `json:"today_orders"`       // 今日新增订单数
	TodayAmount       float64 `json:"today_amount"`       // 今日确认到账金额
	MonthOrders       int64   `json:"month_orders"`       // 本月新增订单数
	MonthAmount       float64 `json:"month_amount"`       // 本月确认到账金额
	PendingPayments   int64   `json:"pending_payments"`   // 待核验收款单数
	PendingDeliveries int64   `json:"pending_deliveries"` // 待交付订单数
	NewLeads          int64   `json:"new_leads"`          // 跟进中线索数
	OverdueLeads      int64   `json:"overdue_leads"`      // 逾期未跟进线索数

	// —— 扩展口径 ——
	PendingDeposit int64   `json:"pending_deposit"` // 待定金订单数
	PendingRetouch int64   `json:"pending_retouch"` // 精修中订单数
	UpcomingShoots int64   `json:"upcoming_shoots"` // 未来待拍摄订单数
	TodayConfirmed float64 `json:"today_confirmed"` // 今日确认到账金额（同 today_amount）
	TodayPending   float64 `json:"today_pending"`   // 今日申报待核验金额
	UnreadNotify   int64   `json:"unread_notify"`   // 当前登录人未读通知数

	// —— 工作台扩展聚合（原型）——
	MonthLeads     int64        `json:"month_leads"`     // 本月新增线索数（成交率分母）
	MonthDealRate  float64      `json:"month_deal_rate"` // 本月成交率 % = 本月新增订单 / 本月新增线索
	AvailableSlots int64        `json:"available_slots"` // 未来 7 天剩余可约时段数
	TodayShoots    []TodayShoot `json:"today_shoots"`    // 今日拍摄列表
	TodoItems      []TodoItem   `json:"todo_items"`      // 待办清单（仅 count > 0）
}

// TodayShoot 今日拍摄条目
type TodayShoot struct {
	ID           int64  `json:"id"`            // 订单ID
	Code         string `json:"code"`          // 订单编号
	CustomerName string `json:"customer_name"` // 客户姓名
	PackageName  string `json:"package_name"`  // 套餐名称
	ShootTime    string `json:"shoot_time"`    // 拍摄时段
	ShootAddress string `json:"shoot_address"` // 拍摄地点
	Photographer string `json:"photographer"`  // 摄影师
	Status       int    `json:"status"`        // 订单状态（2-待拍摄 3-拍摄中）
}

// TodoItem 待办条目
type TodoItem struct {
	Key   string `json:"key"`   // 唯一标识（前端 v-for key）
	Label string `json:"label"` // 展示文案
	Count int64  `json:"count"` // 待处理数量
	Route string `json:"route"` // 点击跳转的前端路由
	Tone  string `json:"tone"`  // 色调：danger / warning / normal
}
