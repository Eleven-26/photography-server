package enum

// 本文件聚合「订单 / 收款 / 退款 / 交付 / 改期」交易链路的状态枚举。
// 原按单状态拆分为 order_status.go / payment_status.go / refund_status.go /
// delivery_stage.go / client_status.go 五个碎片文件（#43 合并）。

// OrderStatus 订单状态
type OrderStatus int

const (
	OrderStatusPendingConfirm  OrderStatus = 0 // 待确认（客户 H5/小程序 预约，待摄影师确认）
	OrderStatusPendingDeposit  OrderStatus = 1 // 待定金
	OrderStatusPendingShoot    OrderStatus = 2 // 待拍摄
	OrderStatusShooting        OrderStatus = 3 // 拍摄中
	OrderStatusRetouching      OrderStatus = 4 // 精修中
	OrderStatusPendingDelivery OrderStatus = 5 // 待交付
	OrderStatusCompleted       OrderStatus = 6 // 已完成
	OrderStatusCancelled       OrderStatus = 7 // 已取消
)

var orderStatusName = map[OrderStatus]string{
	OrderStatusPendingConfirm:  "待确认",
	OrderStatusPendingDeposit:  "待定金",
	OrderStatusPendingShoot:    "待拍摄",
	OrderStatusShooting:        "拍摄中",
	OrderStatusRetouching:      "精修中",
	OrderStatusPendingDelivery: "待交付",
	OrderStatusCompleted:       "已完成",
	OrderStatusCancelled:       "已取消",
}

func OrderStatusName(status OrderStatus) string {
	if name, ok := orderStatusName[status]; ok {
		return name
	}
	return "未知"
}

// PaymentStatus 支付状态
type PaymentStatus int

const (
	PaymentStatusPending   PaymentStatus = 1 // 待核验
	PaymentStatusConfirmed PaymentStatus = 2 // 已确认
	PaymentStatusUnpaid    PaymentStatus = 3 // 待支付
	PaymentStatusRefunded  PaymentStatus = 4 // 已退款
)

var paymentStatusName = map[PaymentStatus]string{
	PaymentStatusPending:   "待核验",
	PaymentStatusConfirmed: "已确认",
	PaymentStatusUnpaid:    "待支付",
	PaymentStatusRefunded:  "已退款",
}

func PaymentStatusName(status PaymentStatus) string {
	if name, ok := paymentStatusName[status]; ok {
		return name
	}
	return "未知"
}

// RefundStatus 退款状态
type RefundStatus int

const (
	RefundStatusApplying RefundStatus = 1 // 申请中
	RefundStatusApproved RefundStatus = 2 // 已通过
	RefundStatusDone     RefundStatus = 3 // 已退款
	RefundStatusRejected RefundStatus = 4 // 已驳回
)

var refundStatusName = map[RefundStatus]string{
	RefundStatusApplying: "申请中",
	RefundStatusApproved: "已通过",
	RefundStatusDone:     "已退款",
	RefundStatusRejected: "已驳回",
}

func RefundStatusName(status RefundStatus) string {
	if name, ok := refundStatusName[status]; ok {
		return name
	}
	return "未知"
}

// DeliveryStage 交付阶段
type DeliveryStage int

const (
	DeliveryStagePendingSamples DeliveryStage = 1 // 待上传样片
	DeliveryStageSelecting      DeliveryStage = 2 // 客户选片中
	DeliveryStageRetouching     DeliveryStage = 3 // 精修进行中
	DeliveryStagePendingConfirm DeliveryStage = 4 // 待确认交付
	DeliveryStageDelivered      DeliveryStage = 5 // 已交付
)

var deliveryStageName = map[DeliveryStage]string{
	DeliveryStagePendingSamples: "待上传样片",
	DeliveryStageSelecting:      "客户选片中",
	DeliveryStageRetouching:     "精修进行中",
	DeliveryStagePendingConfirm: "待确认交付",
	DeliveryStageDelivered:      "已交付",
}

func DeliveryStageName(stage DeliveryStage) string {
	if name, ok := deliveryStageName[stage]; ok {
		return name
	}
	return "未知"
}

// DeliveryFeedbackStatus 客户修图反馈处理状态（biz_delivery_item.feedback_status）
const (
	FeedbackNone    = 0 // 无反馈
	FeedbackPending = 1 // 待处理
	FeedbackHandled = 2 // 已处理
)

// RescheduleStatus 改期单状态
type RescheduleStatus int

const (
	RescheduleStatusPending   RescheduleStatus = 1 // 待确认
	RescheduleStatusApproved  RescheduleStatus = 2 // 已同意
	RescheduleStatusRejected  RescheduleStatus = 3 // 已拒绝
	RescheduleStatusCancelled RescheduleStatus = 4 // 已取消
)

var rescheduleStatusName = map[RescheduleStatus]string{
	RescheduleStatusPending:   "待确认",
	RescheduleStatusApproved:  "已同意",
	RescheduleStatusRejected:  "已拒绝",
	RescheduleStatusCancelled: "已取消",
}

func RescheduleStatusName(s RescheduleStatus) string {
	if n, ok := rescheduleStatusName[s]; ok {
		return n
	}
	return "未知"
}

// RescheduleFeeType 改期费用类型
type RescheduleFeeType int

const (
	RescheduleFeeFree      RescheduleFeeType = 1 // 免费
	RescheduleFeeCharged   RescheduleFeeType = 2 // 收调度费
	RescheduleFeeForbidden RescheduleFeeType = 3 // 不可改期
)
