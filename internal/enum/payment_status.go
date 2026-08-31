package enum

// PaymentStatus 支付状态
type PaymentStatus int

const (
	PaymentStatusPending   PaymentStatus = 1 // 待确认
	PaymentStatusConfirmed PaymentStatus = 2 // 已确认
	PaymentStatusRefunded  PaymentStatus = 3 // 已退款
)

var paymentStatusName = map[PaymentStatus]string{
	PaymentStatusPending:   "待确认",
	PaymentStatusConfirmed: "已确认",
	PaymentStatusRefunded:  "已退款",
}

func PaymentStatusName(status PaymentStatus) string {
	if name, ok := paymentStatusName[status]; ok {
		return name
	}
	return "未知"
}
