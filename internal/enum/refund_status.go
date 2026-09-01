package enum

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
