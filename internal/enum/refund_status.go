package enum

// RefundStatus 退款状态
type RefundStatus int

const (
	RefundStatusPending  RefundStatus = 1 // 待审核
	RefundStatusApproved RefundStatus = 2 // 已通过
	RefundStatusRejected RefundStatus = 3 // 已驳回
	RefundStatusRefunded RefundStatus = 4 // 已退款
)

var refundStatusName = map[RefundStatus]string{
	RefundStatusPending:  "待审核",
	RefundStatusApproved: "已通过",
	RefundStatusRejected: "已驳回",
	RefundStatusRefunded: "已退款",
}

func RefundStatusName(status RefundStatus) string {
	if name, ok := refundStatusName[status]; ok {
		return name
	}
	return "未知"
}
