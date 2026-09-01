package dto

// ======================== 请求 ========================

// CalendarBlockReq 锁定档期
type CalendarBlockReq struct {
	StoreID        int64  `json:"store_id"`                      // 门店ID
	OrderID        int64  `json:"order_id"`                      // 订单ID
	CustomerID     int64  `json:"customer_id"`                   // 客户ID
	CustomerName   string `json:"customer_name"`                 // 客户姓名
	Date           string `json:"date" binding:"required"`       // 日期，格式：2006-01-02
	TimeRange      string `json:"time_range" binding:"required"` // 时段，如：09:00-12:00
	ProjectType    string `json:"project_type"`                  // 项目类型
	PhotographerID int64  `json:"photographer_id"`               // 摄影师ID
	Photographer   string `json:"photographer"`                  // 摄影师姓名
	Remark         string `json:"remark"`                        // 备注
}

// ======================== 响应 ========================
