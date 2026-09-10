package enum

// 本文件聚合「通知 / 操作日志 / 上传」系统级枚举。
// 原拆分为 notification.go / log_module.go / upload_type.go（#43 合并）。

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

// LogModule 操作日志模块
type LogModule int

const (
	LogModuleCustomer LogModule = 1
	LogModuleLead     LogModule = 2
	LogModuleOrder    LogModule = 3
	LogModulePackage  LogModule = 4
	LogModuleDelivery LogModule = 5
	LogModuleCalendar LogModule = 6
	LogModuleFinance  LogModule = 7
	LogModuleSettings LogModule = 8
)

var logModuleName = map[LogModule]string{
	LogModuleCustomer: "客户",
	LogModuleLead:     "线索",
	LogModuleOrder:    "订单",
	LogModulePackage:  "套餐",
	LogModuleDelivery: "交付",
	LogModuleCalendar: "档期",
	LogModuleFinance:  "财务",
	LogModuleSettings: "设置",
}

func LogModuleName(module LogModule) string {
	if name, ok := logModuleName[module]; ok {
		return name
	}
	return "未知"
}

// UploadType 上传文件类型
type UploadType int

const (
	UploadTypeImage UploadType = 1 // 图片
	UploadTypeVideo UploadType = 2 // 视频
	UploadTypeFile  UploadType = 3 // 文件
)

var uploadTypeName = map[UploadType]string{
	UploadTypeImage: "图片",
	UploadTypeVideo: "视频",
	UploadTypeFile:  "文件",
}

func UploadTypeName(typ UploadType) string {
	if name, ok := uploadTypeName[typ]; ok {
		return name
	}
	return "未知"
}
