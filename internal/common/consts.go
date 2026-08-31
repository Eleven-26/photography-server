package common

// 订单状态
const (
	OrderStatusPendingDeposit  = "pending_deposit"  // 待定金
	OrderStatusPendingShoot    = "pending_shoot"    // 待拍摄
	OrderStatusShooting        = "shooting"         // 拍摄中
	OrderStatusRetouching      = "retouching"       // 精修中
	OrderStatusPendingDelivery = "pending_delivery" // 待交付
	OrderStatusCompleted       = "completed"        // 已完成
	OrderStatusCancelled       = "cancelled"        // 已取消
)

// 交付阶段
const (
	DeliveryStagePendingSamples = "pending_samples" // 待上传样片
	DeliveryStageSelecting      = "selecting"       // 客户选片中
	DeliveryStageRetouching     = "retouching"      // 精修进行中
	DeliveryStagePendingConfirm = "pending_confirm" // 待确认交付
	DeliveryStageDelivered      = "delivered"       // 已交付
)

// 线索状态
const (
	LeadStatusNew       = "new"       // 新线索
	LeadStatusFollowing = "following" // 跟进中
	LeadStatusQuoting   = "quoting"   // 报价中
	LeadStatusConverted = "converted" // 已转化
	LeadStatusLost      = "lost"      // 已流失
)

// 客户等级
const (
	CustomerLevelNormal = "normal" // 普通
	CustomerLevelSilver = "silver" // 银卡
	CustomerLevelGold   = "gold"   // 金卡
	CustomerLevelSVIP   = "svip"   // SVIP
)

// 支付状态
const (
	PaymentStatusPending   = "pending"   // 待确认
	PaymentStatusConfirmed = "confirmed" // 已确认
	PaymentStatusRefunded  = "refunded"  // 已退款
)

// 退款状态
const (
	RefundStatusPending  = "pending"  // 待审核
	RefundStatusApproved = "approved" // 已通过
	RefundStatusRejected = "rejected" // 已驳回
	RefundStatusRefunded = "refunded" // 已退款
)

// 操作日志模块
const (
	ModuleCustomer = "customer"
	ModuleLead     = "lead"
	ModuleOrder    = "order"
	ModulePackage  = "package"
	ModuleDelivery = "delivery"
	ModuleCalendar = "calendar"
	ModuleFinance  = "finance"
	ModuleSettings = "settings"
)
