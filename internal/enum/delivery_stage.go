package enum

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
