package model

// Order 订单
type Order struct {
	TenantBase
	Code           string  `gorm:"column:code;size:30;not null;uniqueIndex:uk_order_code,priority:1;comment:订单编号 SL-xxx" json:"code"`
	StoreID        int64   `gorm:"column:store_id;index;comment:所属门店ID" json:"store_id"`
	CustomerID     int64   `gorm:"column:customer_id;index;comment:客户ID" json:"customer_id"`
	CustomerName   string  `gorm:"column:customer_name;size:50;comment:客户姓名(快照)" json:"customer_name"`
	CustomerMobile string  `gorm:"column:customer_mobile;size:20;comment:客户手机号(快照)" json:"customer_mobile"`
	LeadID         int64   `gorm:"column:lead_id;index;comment:来源线索ID" json:"lead_id"`
	QuoteID        int64   `gorm:"column:quote_id;index;comment:来源报价单ID" json:"quote_id"`
	PackageID      int64   `gorm:"column:package_id;index;comment:套餐ID" json:"package_id"`
	PackageName    string  `gorm:"column:package_name;size:100;comment:套餐名称(快照)" json:"package_name"`
	PackageVersion int     `gorm:"column:package_version;default:1;comment:下单套餐版本号" json:"package_version"`
	BasePrice      float64 `gorm:"column:base_price;type:decimal(12,2);comment:基础套餐价(快照)" json:"base_price"`
	AddonAmount    float64 `gorm:"column:addon_amount;type:decimal(12,2);default:0;comment:加选精修金额(全部进尾款)" json:"addon_amount"`
	DepositAmt     float64 `gorm:"column:deposit_amt;type:decimal(12,2);default:0;comment:定金金额(基础价×定金比例)" json:"deposit_amt"`
	FinalAmt       float64 `gorm:"column:final_amt;type:decimal(12,2);default:0;comment:尾款金额(基础价-定金+加选)" json:"final_amt"`
	TotalAmt       float64 `gorm:"column:total_amt;type:decimal(12,2);default:0;comment:订单总额" json:"total_amt"`
	PaidAmt        float64 `gorm:"column:paid_amt;type:decimal(12,2);default:0;comment:已收款金额" json:"paid_amt"`
	RefundAmt      float64 `gorm:"column:refund_amt;type:decimal(12,2);default:0;comment:已退款金额" json:"refund_amt"`
	Status         string  `gorm:"column:status;size:20;default:pending_deposit;comment:订单状态" json:"status"`
	PaymentStatus  string  `gorm:"column:payment_status;size:20;default:pending;comment:支付状态 pending-待核验 confirmed-已确认 unpaid-待支付 refunded-已退款" json:"payment_status"`
	ShootDate      *string `gorm:"column:shoot_date;comment:拍摄日期" json:"shoot_date"`
	ShootTime      string  `gorm:"column:shoot_time;size:20;comment:拍摄时间段" json:"shoot_time"`
	ShootAddress   string  `gorm:"column:shoot_address;size:200;comment:拍摄地点" json:"shoot_address"`
	PhotographerID int64   `gorm:"column:photographer_id;index;comment:摄影师ID" json:"photographer_id"`
	Photographer   string  `gorm:"column:photographer;size:50;comment:摄影师姓名" json:"photographer"`
	Remark         string  `gorm:"column:remark;size:500;comment:备注" json:"remark"`
	CancelReason   string  `gorm:"column:cancel_reason;size:200;comment:取消原因" json:"cancel_reason"`
	FinishedAt     *string `gorm:"column:finished_at;comment:完成时间" json:"finished_at"`
	OwnerID        int64   `gorm:"column:owner_id;index;comment:负责人(销售)ID" json:"owner_id"`
}

func (Order) TableName() string { return "biz_order" }

// OrderPayment 收款记录
type OrderPayment struct {
	TenantBase
	OrderID      int64   `gorm:"column:order_id;index;comment:订单ID" json:"order_id"`
	Code         string  `gorm:"column:code;size:20;not null;uniqueIndex:uk_payment_code,priority:1;comment:收款单号 PM-xxx" json:"code"`
	CustomerID   int64   `gorm:"column:customer_id;index;comment:客户ID" json:"customer_id"`
	Type         string  `gorm:"column:type;size:20;comment:收款类型 deposit-定金 final-尾款 addon-加选" json:"type"`
	Amount       float64 `gorm:"column:amount;type:decimal(12,2);not null;comment:收款金额" json:"amount"`
	MethodID     int64   `gorm:"column:method_id;comment:收款方式ID" json:"method_id"`
	MethodName   string  `gorm:"column:method_name;size:50;comment:收款方式名称(快照)" json:"method_name"`
	Status       string  `gorm:"column:status;size:20;default:pending;comment:状态 pending-待核验 confirmed-已确认 refunded-已退款" json:"status"`
	PaidAt       *string `gorm:"column:paid_at;comment:收款时间" json:"paid_at"`
	Voucher      string  `gorm:"column:voucher;size:500;comment:收款凭证图片" json:"voucher"`
	OperatorID   int64   `gorm:"column:operator_id;comment:收款操作人ID" json:"operator_id"`
	OperatorName string  `gorm:"column:operator_name;size:50;comment:收款操作人" json:"operator_name"`
	Remark       string  `gorm:"column:remark;size:200;comment:备注" json:"remark"`
}

func (OrderPayment) TableName() string { return "biz_order_payment" }

// OrderRefund 退款记录
type OrderRefund struct {
	TenantBase
	OrderID     int64   `gorm:"column:order_id;index;comment:订单ID" json:"order_id"`
	Code        string  `gorm:"column:code;size:20;not null;uniqueIndex:uk_refund_code,priority:1;comment:退款单号 RF-xxx" json:"code"`
	CustomerID  int64   `gorm:"column:customer_id;index;comment:客户ID" json:"customer_id"`
	Amount      float64 `gorm:"column:amount;type:decimal(12,2);not null;comment:退款金额" json:"amount"`
	Reason      string  `gorm:"column:reason;size:200;comment:退款原因" json:"reason"`
	RefundRule  string  `gorm:"column:refund_rule;size:50;comment:退款规则档位(按拍摄前小时)" json:"refund_rule"`
	Status      string  `gorm:"column:status;size:20;default:applying;comment:状态 applying-申请中 approved-已通过 done-已退款 rejected-已驳回" json:"status"`
	ApplyBy     int64   `gorm:"column:apply_by;comment:申请人ID" json:"apply_by"`
	ApplyName   string  `gorm:"column:apply_name;size:50;comment:申请人" json:"apply_name"`
	AuditBy     int64   `gorm:"column:audit_by;comment:审核人ID" json:"audit_by"`
	AuditAt     *string `gorm:"column:audit_at;comment:审核时间" json:"audit_at"`
	AuditRemark string  `gorm:"column:audit_remark;size:200;comment:审核备注" json:"audit_remark"`
	RefundAt    *string `gorm:"column:refund_at;comment:退款时间" json:"refund_at"`
}

func (OrderRefund) TableName() string { return "biz_order_refund" }

// OrderLog 订单操作日志
type OrderLog struct {
	TenantBase
	OrderID      int64  `gorm:"column:order_id;index;comment:订单ID" json:"order_id"`
	Action       string `gorm:"column:action;size:50;comment:操作动作" json:"action"`
	FromStatus   string `gorm:"column:from_status;size:20;comment:原状态" json:"from_status"`
	ToStatus     string `gorm:"column:to_status;size:20;comment:新状态" json:"to_status"`
	Content      string `gorm:"column:content;size:500;comment:操作内容" json:"content"`
	OperatorID   int64  `gorm:"column:operator_id;comment:操作人ID" json:"operator_id"`
	OperatorName string `gorm:"column:operator_name;size:50;comment:操作人" json:"operator_name"`
}

func (OrderLog) TableName() string { return "biz_order_log" }
