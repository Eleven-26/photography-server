package enum

// OrderStatus 订单状态
type OrderStatus struct {
	Code int    // 枚举编码
	Name string // 枚举信息
}

var (
	OrderStatusPendingDeposit  = OrderStatus{Code: 1, Name: "待定金"}
	OrderStatusPendingShoot    = OrderStatus{Code: 2, Name: "待拍摄"}
	OrderStatusShooting        = OrderStatus{Code: 3, Name: "拍摄中"}
	OrderStatusRetouching      = OrderStatus{Code: 4, Name: "精修中"}
	OrderStatusPendingDelivery = OrderStatus{Code: 5, Name: "待交付"}
	OrderStatusCompleted       = OrderStatus{Code: 6, Name: "已完成"}
	OrderStatusCancelled       = OrderStatus{Code: 7, Name: "已取消"}
)

// 枚举实例的切片
var enumList = []OrderStatus{
	OrderStatusPendingDeposit,
	OrderStatusPendingShoot,
	OrderStatusShooting,
	OrderStatusRetouching,
	OrderStatusPendingDelivery,
	OrderStatusCompleted,
	OrderStatusCancelled,
}

func OrderStatusName(code int) string {
	for _, e := range enumList {
		if e.Code == code {
			return e.Name
		}
	}
	return ""
}

/**
---- 另一种写法

// OrderStatus 订单状态
type OrderStatus struct {
	Code int    // 枚举编码
	Name string // 枚举信息
}

var (
	OrderStatusPendingDeposit  = OrderStatus{Code: 1, Name: "待定金"}
	OrderStatusPendingShoot    = OrderStatus{Code: 2, Name: "待拍摄"}
	OrderStatusShooting        = OrderStatus{Code: 3, Name: "拍摄中"}
	OrderStatusRetouching      = OrderStatus{Code: 4, Name: "精修中"}
	OrderStatusPendingDelivery = OrderStatus{Code: 5, Name: "待交付"}
	OrderStatusCompleted       = OrderStatus{Code: 6, Name: "已完成"}
	OrderStatusCancelled       = OrderStatus{Code: 7, Name: "已取消"}
)

// 枚举实例的切片
var enumList = []OrderStatus{
	OrderStatusPendingDeposit,
	OrderStatusPendingShoot,
	OrderStatusShooting,
	OrderStatusRetouching,
	OrderStatusPendingDelivery,
	OrderStatusCompleted,
	OrderStatusCancelled,
}

func OrderStatusName(code int) string {
	for _, e := range enumList {
		if e.Code == code {
			return e.Name
		}
	}
	return ""
}
*/
