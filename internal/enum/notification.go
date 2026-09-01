package enum

// NotificationType 通知类型
type NotificationType int

const (
	NotificationTypeOrder   NotificationType = 1 // 订单
	NotificationTypeFinance NotificationType = 2 // 财务
	NotificationTypeSystem  NotificationType = 3 // 系统
)

var notificationTypeName = map[NotificationType]string{
	NotificationTypeOrder:   "订单",
	NotificationTypeFinance: "财务",
	NotificationTypeSystem:  "系统",
}

func NotificationTypeName(typ NotificationType) string {
	if name, ok := notificationTypeName[typ]; ok {
		return name
	}
	return "未知"
}

// NotificationReadStatus 通知已读状态
type NotificationReadStatus int

const (
	NotificationUnread NotificationReadStatus = 0 // 未读
	NotificationRead   NotificationReadStatus = 1 // 已读
)
