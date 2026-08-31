package enum

// OrderStatus 订单状态
type OrderStatus int

const (
	OrderStatusPendingDeposit  OrderStatus = 1 // 待定金
	OrderStatusPendingShoot    OrderStatus = 2 // 待拍摄
	OrderStatusShooting        OrderStatus = 3 // 拍摄中
	OrderStatusRetouching      OrderStatus = 4 // 精修中
	OrderStatusPendingDelivery OrderStatus = 5 // 待交付
	OrderStatusCompleted       OrderStatus = 6 // 已完成
	OrderStatusCancelled       OrderStatus = 7 // 已取消
)

var orderStatusName = map[OrderStatus]string{
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
