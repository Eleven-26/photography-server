package contract

import "photography-server/internal/model"

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
	ID             int64  `json:"id"`              // 订单ID（body 主键）
	ShootDate      string `json:"shoot_date"`      // 拍摄日期
	ShootTime      string `json:"shoot_time"`      // 拍摄时段
	ShootAddress   string `json:"shoot_address"`   // 拍摄地址
	PhotographerID int64  `json:"photographer_id"` // 摄影师ID
	Photographer   string `json:"photographer"`    // 摄影师姓名
	Remark         string `json:"remark"`          // 备注
}

// OrderStatusReq 订单状态流转
type OrderStatusReq struct {
	Status  int    `json:"status" binding:"required"` // 目标状态
	Content string `json:"content"`                   // 状态变更说明
}

// OrderAddonReq 订单加项创建/更新
type OrderAddonReq struct {
	Name      string  `json:"name" binding:"required"` // 加项名称
	Category  string  `json:"category"`                // 分类 makeup-妆造 urgency-时效 service-服务 retouch-精修
	Price     float64 `json:"price"`                   // 单价
	Qty       int     `json:"qty"`                     // 数量（<=0 按 1 计）
	Confirmed int     `json:"confirmed"`               // 客户是否确认 0-待确认 1-已确认
	Remark    string  `json:"remark"`                  // 备注
}

// RescheduleApplyReq 管理端发起改期（摄影师/管理员代客户申请）
type RescheduleApplyReq struct {
	NewDate     string `json:"new_date" binding:"required"` // 新拍摄日期
	NewTime     string `json:"new_time" binding:"required"` // 新拍摄时段
	ReasonLabel string `json:"reason_label"`                // 改期原因分类
	Reason      string `json:"reason"`                      // 改期原因说明
}

// OrderDetail 订单详情。各子项为具体类型而非 interface{}：
// 字段改名/缺失会在编译期暴露，不再出现「前后端字段静默漂移」。
type OrderDetail struct {
	Order    *model.Order         `json:"order"`    // 订单信息
	Payments []model.OrderPayment `json:"payments"` // 收款记录
	Refunds  []model.OrderRefund  `json:"refunds"`  // 退款记录
	Logs     []model.OrderLog     `json:"logs"`     // 操作日志
	Delivery *model.Delivery      `json:"delivery"` // 交付信息（未创建交付单时为 null）
	// AllowedTransitions 当前状态允许流转到的目标状态（领域状态机输出）。
	// 前端据此渲染「阶段推进」按钮，避免在客户端重复实现状态机导致规则漂移。
	AllowedTransitions []int `json:"allowed_transitions"`
}

// PaymentCreateReq 创建收款
type PaymentCreateReq struct {
	OrderID  int64   `json:"order_id"`                  // 订单ID（body 主键）
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

// OrderListReq 订单列表查询（body：分页 + 过滤）。
//
// 仅用于 Swagger 文档：handler 通过 params 中间件从 JSON body 读取这些字段。
type OrderListReq struct {
	Page       int    `json:"page"`        // 页码，默认 1
	PageSize   int    `json:"page_size"`   // 每页条数，默认 20，上限 200
	Status     string `json:"status"`      // 订单状态过滤
	CustomerID int64  `json:"customer_id"` // 客户ID过滤
}

// OrderCancelReq 取消订单请求
type OrderCancelReq struct {
	Reason string `json:"reason"` // 取消原因
}

// AuditReq 审批类请求（退款审批 / 改期审批共用）。
// approved 用指针：区分「未传」与「驳回(false)」；为 nil 时返回参数错误。
type AuditReq struct {
	Approved *bool  `json:"approved"` // 审批结论 true-通过 false-驳回
	Remark   string `json:"remark"`   // 审批备注
}

// OrderAddonCreateReq 创建订单加项请求（order_id 与加项明细同 body）
type OrderAddonCreateReq struct {
	OrderID int64 `json:"order_id"` // 订单ID（body 主键）
	OrderAddonReq
}

// OrderAddonUpdateReq 更新订单加项请求（id 与加项明细同 body）
type OrderAddonUpdateReq struct {
	ID int64 `json:"id"` // 加项ID（body 主键）
	OrderAddonReq
}
