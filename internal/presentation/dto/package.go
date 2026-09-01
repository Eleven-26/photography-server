package dto

// ======================== 请求 ========================

// PackageReq 创建/更新套餐
type PackageReq struct {
	StoreID        int64   `json:"store_id"`
	Name           string  `json:"name" binding:"required"`
	Cover          string  `json:"cover"`
	Category       string  `json:"category"`
	BasePrice      float64 `json:"base_price" binding:"required"`
	DepositRate    float64 `json:"deposit_rate"`
	PhotosIncluded int     `json:"photos_included"`
	ShootHours     float64 `json:"shoot_hours"`
	ContentDesc    string  `json:"content_desc"`
	AddonUnitPrice float64 `json:"addon_unit_price"`
	Status         string  `json:"status"`
}

// ======================== 响应 ========================
