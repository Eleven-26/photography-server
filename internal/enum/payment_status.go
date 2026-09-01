package enum

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
