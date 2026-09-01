package dto

// ======================== 请求 ========================

// OrderCreateReq 创建订单
type OrderCreateReq struct {
	CustomerID     int64   `json:"customer_id"`                   // 客户ID
	LeadID         int64   `json:"lead_id"`                       // 线索ID（从线索转化时传）
	QuoteID        int64   `json:"quote_id"`                      // 报价单ID
	PackageID      int64   `json:"package_id" binding:"required"` // 套餐ID
	AddonAmount    float64 `json:"addon_amount"`                  // 加选金额
	ShootDate      string  `json:"shoot_date"`                    // 拍摄日期
	ShootTime      string  `json:"shoot_time"`                    // 拍摄时段
	ShootAddress   string  `json:"shoot_address"`                 // 拍摄地址
	PhotographerID int64   `json:"photographer_id"`               // 摄影师ID
	Photographer   string  `json:"photographer"`                  // 摄影师姓名
	Remark         string  `json:"remark"`                        // 备注
	OwnerID        int64   `json:"owner_id"`                      // 负责人ID
}

// OrderUpdateReq 更新订单
type OrderUpdateReq struct {
	ShootDate      string `json:"shoot_date"`      // 拍摄日期
	ShootTime      string `json:"shoot_time"`      // 拍摄时段
	ShootAddress   string `json:"shoot_address"`   // 拍摄地址
	PhotographerID int64  `json:"photographer_id"` // 摄影师ID
	Photographer   string `json:"photographer"`    // 摄影师姓名
	Remark         string `json:"remark"`          // 备注
}

// OrderStatusReq 订单状态流转
type OrderStatusReq struct {
	Status  string `json:"status" binding:"required"` // 目标状态
	Content string `json:"content"`                   // 状态变更说明
}

// ======================== 响应 ========================

// OrderDetail 订单详情
type OrderDetail struct {
	Order    interface{} `json:"order"`    // 订单信息
	Payments interface{} `json:"payments"` // 收款记录
	Refunds  interface{} `json:"refunds"`  // 退款记录
	Logs     interface{} `json:"logs"`     // 操作日志
	Delivery interface{} `json:"delivery"` // 交付信息
}

// PaymentCreateReq 创建收款
type PaymentCreateReq struct {
	Type     string  `json:"type" binding:"required"`   // 类型: deposit-定金, final-尾款, addon-加选
	Amount   float64 `json:"amount" binding:"required"` // 金额
	MethodID int64   `json:"method_id"`                 // 收款方式ID
	PaidAt   string  `json:"paid_at"`                   // 付款时间
	Voucher  string  `json:"voucher"`                   // 凭证URL
	Remark   string  `json:"remark"`                    // 备注
}

// RefundCreateReq 申请退款
type RefundCreateReq struct {
	Reason string  `json:"reason"` // 退款原因
	Amount float64 `json:"amount"` // 退款金额，为空时按规则自动计算
}

// ======================== 响应 ========================
