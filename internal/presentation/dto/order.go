package dto

// ======================== 请求 ========================

// OrderCreateReq 创建订单
type OrderCreateReq struct {
	CustomerID     int64   `json:"customer_id"`
	LeadID         int64   `json:"lead_id"`
	QuoteID        int64   `json:"quote_id"`
	PackageID      int64   `json:"package_id" binding:"required"`
	AddonAmount    float64 `json:"addon_amount"`
	ShootDate      string  `json:"shoot_date"`
	ShootTime      string  `json:"shoot_time"`
	ShootAddress   string  `json:"shoot_address"`
	PhotographerID int64   `json:"photographer_id"`
	Photographer   string  `json:"photographer"`
	Remark         string  `json:"remark"`
	OwnerID        int64   `json:"owner_id"`
}

// OrderUpdateReq 更新订单
type OrderUpdateReq struct {
	ShootDate      string `json:"shoot_date"`
	ShootTime      string `json:"shoot_time"`
	ShootAddress   string `json:"shoot_address"`
	PhotographerID int64  `json:"photographer_id"`
	Photographer   string `json:"photographer"`
	Remark         string `json:"remark"`
}

// OrderStatusReq 订单状态流转
type OrderStatusReq struct {
	Status  string `json:"status" binding:"required"`
	Content string `json:"content"`
}

// ======================== 响应 ========================

// OrderDetail 订单详情
type OrderDetail struct {
	Order    interface{} `json:"order"`
	Payments interface{} `json:"payments"`
	Refunds  interface{} `json:"refunds"`
	Logs     interface{} `json:"logs"`
	Delivery interface{} `json:"delivery"`
}

// PaymentCreateReq 创建收款
type PaymentCreateReq struct {
	Type     string  `json:"type" binding:"required"` // deposit-定金 final-尾款 addon-加选
	Amount   float64 `json:"amount" binding:"required"`
	MethodID int64   `json:"method_id"`
	PaidAt   string  `json:"paid_at"`
	Voucher  string  `json:"voucher"`
	Remark   string  `json:"remark"`
}

// RefundCreateReq 申请退款
type RefundCreateReq struct {
	Reason string  `json:"reason"`
	Amount float64 `json:"amount"` // 为空时按规则自动计算
}
