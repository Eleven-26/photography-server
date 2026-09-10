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

// NotificationReceiver 通知接收人类型。
// 员工与客户是两套独立的 ID 空间（sys_user.id / crm_customer.id），
// 查询与已读回写必须同时按 receiver_type + receiver_id 过滤，否则「客户 5」会看到「员工 5」的通知。
// 存量数据由列默认值 1（员工）覆盖。
const (
	NotificationReceiverStaff    = 1 // 员工（sys_user.id）
	NotificationReceiverCustomer = 2 // 客户（crm_customer.id）
)
