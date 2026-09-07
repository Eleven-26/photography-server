package enum

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

// CustomRequestStatus 定制需求状态
type CustomRequestStatus int

const (
	CustomRequestPending   CustomRequestStatus = 1 // 待处理
	CustomRequestResponded CustomRequestStatus = 2 // 已响应
	CustomRequestClosed    CustomRequestStatus = 3 // 已关闭
)

// BriefItemStatus AI 简报项状态
type BriefItemStatus int

const (
	BriefItemPending   BriefItemStatus = 1 // 待追问
	BriefItemSent      BriefItemStatus = 2 // 已发送
	BriefItemConfirmed BriefItemStatus = 3 // 已确认
)
