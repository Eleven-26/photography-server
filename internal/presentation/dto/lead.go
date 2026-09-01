package dto

// ======================== 请求 ========================

// LeadCreateReq 创建线索
type LeadCreateReq struct {
	StoreID     int64   `json:"store_id"`
	Name        string  `json:"name" binding:"required"`
	Mobile      string  `json:"mobile"`
	Source      string  `json:"source"`
	ProjectType string  `json:"project_type"`
	BudgetMin   float64 `json:"budget_min"`
	BudgetMax   float64 `json:"budget_max"`
	ShootDate   string  `json:"shoot_date"`
	Remark      string  `json:"remark"`
	OwnerID     int64   `json:"owner_id"`
}

// LeadUpdateReq 更新线索
type LeadUpdateReq struct {
	StoreID     int64   `json:"store_id"`
	Name        string  `json:"name"`
	Mobile      string  `json:"mobile"`
	Source      string  `json:"source"`
	ProjectType string  `json:"project_type"`
	BudgetMin   float64 `json:"budget_min"`
	BudgetMax   float64 `json:"budget_max"`
	Status      string  `json:"status"`
	ShootDate   string  `json:"shoot_date"`
	Remark      string  `json:"remark"`
	OwnerID     int64   `json:"owner_id"`
}

// LeadFollowReq 跟进线索
type LeadFollowReq struct {
	Remark string `json:"remark"`
}

// QuoteCreateReq 创建报价单
type QuoteCreateReq struct {
	PackageID  int64   `json:"package_id" binding:"required"`
	Title      string  `json:"title"`
	AddonPrice float64 `json:"addon_price"`
	ShootDate  string  `json:"shoot_date"`
	Remark     string  `json:"remark"`
}

// ======================== 响应 ========================
