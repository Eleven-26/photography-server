package model

// 全局业务状态常量

// 订单状态
const (
	OrderStatusPendingDeposit   = "pending_deposit"   // 待定金
	OrderStatusScheduled        = "scheduled"         // 待拍摄
	OrderStatusShooting         = "shooting"          // 拍摄中
	OrderStatusRetouching       = "retouching"        // 精修中
	OrderStatusAwaitingDelivery = "awaiting_delivery" // 待交付
	OrderStatusCompleted        = "completed"         // 已完成
	OrderStatusCancelled        = "cancelled"         // 已取消
)

// 支付状态
const (
	PaymentStatusPending   = "pending"   // 待核验
	PaymentStatusConfirmed = "confirmed" // 已确认
	PaymentStatusUnpaid    = "unpaid"    // 待支付
	PaymentStatusRefunded  = "refunded"  // 已退款
)

// 退款状态
const (
	RefundStatusApplying = "applying" // 申请中
	RefundStatusApproved = "approved" // 已通过
	RefundStatusDone     = "done"     // 已退款
	RefundStatusRejected = "rejected" // 已驳回
)

// 线索状态
const (
	LeadStatusPending   = "pending"   // 待回复
	LeadStatusQuoting   = "quoting"   // 待报价
	LeadStatusQuoted    = "quoted"    // 已报价
	LeadStatusConfirmed = "confirmed" // 待确认/已成交
	LeadStatusLose      = "lose"      // 已流失
)

// 报价单状态
const (
	QuoteStatusDraft     = "draft"     // 草稿
	QuoteStatusSent      = "sent"      // 已发送
	QuoteStatusAccepted  = "accepted"  // 已接受
	QuoteStatusRejected  = "rejected"  // 已拒绝
	QuoteStatusConverted = "converted" // 已成交
)

// 套餐状态
const (
	PackageStatusActive  = "active"  // 已上架
	PackageStatusDraft   = "draft"   // 草稿
	PackageStatusOffline = "offline" // 已下线
)

// 交付(选片精修)阶段
const (
	DeliveryStageUploadPending = "upload_pending" // 待上传样片
	DeliveryStageSelecting     = "selecting"      // 客户选片中
	DeliveryStageRetouching    = "retouching"     // 精修进行中
	DeliveryStageDeliverReady  = "deliver_ready"  // 待确认交付
	DeliveryStageCompleted     = "completed"      // 已交付
)

// 档期锁定状态
const (
	BlockStatusLocked    = "locked"    // 已锁定
	BlockStatusCancelled = "cancelled" // 已取消
)

// 客户等级
const (
	CustomerLevelNormal   = "normal"   // 普通
	CustomerLevelGold     = "gold"     // 黄金
	CustomerLevelPlatinum = "platinum" // 铂金
	CustomerLevelDiamond  = "diamond"  // 钻石
)

// 客户状态
const (
	CustomerStatusPotential = "potential" // 潜在
	CustomerStatusActive    = "active"    // 活跃
	CustomerStatusInactive  = "inactive"  // 流失/沉睡
)

// 作品集状态
const (
	AssetStatusDraft     = "draft"     // 草稿
	AssetStatusPublished = "published" // 已发布
)

// 上传文件类型
const (
	UploadTypeImage = "image" // 图片
	UploadTypeVideo = "video" // 视频
	UploadTypeFile  = "file"  // 文件
)

// 通知类型
const (
	NotificationTypeOrder   = "order"   // 订单
	NotificationTypeFinance = "finance" // 财务
	NotificationTypeSystem  = "system"  // 系统
)

// 通知是否已读
const (
	NotificationUnread = 0 // 未读
	NotificationRead   = 1 // 已读
)
