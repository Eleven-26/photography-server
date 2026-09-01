package dto

// ======================== 请求 ========================

// CalendarBlockReq 锁定档期
type CalendarBlockReq struct {
	StoreID        int64  `json:"store_id"`
	OrderID        int64  `json:"order_id"`
	CustomerID     int64  `json:"customer_id"`
	CustomerName   string `json:"customer_name"`
	Date           string `json:"date" binding:"required"`
	TimeRange      string `json:"time_range" binding:"required"`
	ProjectType    string `json:"project_type"`
	PhotographerID int64  `json:"photographer_id"`
	Photographer   string `json:"photographer"`
	Remark         string `json:"remark"`
}

// ======================== 响应 ========================
