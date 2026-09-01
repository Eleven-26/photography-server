package dto

// ======================== 请求 ========================

// LeadCreateReq 创建线索
type LeadCreateReq struct {
	StoreID     int64   `json:"store_id"`                // 门店ID
	Name        string  `json:"name" binding:"required"` // 客户姓名
	Mobile      string  `json:"mobile"`                  // 手机号
	Source      string  `json:"source"`                  // 来源
	ProjectType string  `json:"project_type"`            // 项目类型
	BudgetMin   float64 `json:"budget_min"`              // 预算下限
	BudgetMax   float64 `json:"budget_max"`              // 预算上限
	ShootDate   string  `json:"shoot_date"`              // 期望拍摄日期
	Remark      string  `json:"remark"`                  // 备注
	OwnerID     int64   `json:"owner_id"`                // 负责人ID
}

// LeadUpdateReq 更新线索
type LeadUpdateReq struct {
	StoreID     int64   `json:"store_id"`     // 门店ID
	Name        string  `json:"name"`         // 客户姓名
	Mobile      string  `json:"mobile"`       // 手机号
	Source      string  `json:"source"`       // 来源
	ProjectType string  `json:"project_type"` // 项目类型
	BudgetMin   float64 `json:"budget_min"`   // 预算下限
	BudgetMax   float64 `json:"budget_max"`   // 预算上限
	Status      string  `json:"status"`       // 状态
	ShootDate   string  `json:"shoot_date"`   // 期望拍摄日期
	Remark      string  `json:"remark"`       // 备注
	OwnerID     int64   `json:"owner_id"`     // 负责人ID
}

// LeadFollowReq 跟进线索
type LeadFollowReq struct {
	Remark string `json:"remark"` // 跟进备注
}

// QuoteCreateReq 创建报价单
type QuoteCreateReq struct {
	PackageID  int64   `json:"package_id" binding:"required"` // 套餐ID
	Title      string  `json:"title"`                         // 报价单标题
	AddonPrice float64 `json:"addon_price"`                   // 加选费用
	ShootDate  string  `json:"shoot_date"`                    // 拍摄日期
	Remark     string  `json:"remark"`                        // 备注
}

// ======================== 响应 ========================
