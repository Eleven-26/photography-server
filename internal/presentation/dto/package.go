package dto

// ======================== 请求 ========================

// PackageReq 创建/更新套餐
type PackageReq struct {
	StoreID        int64   `json:"store_id"`                      // 门店ID
	Name           string  `json:"name" binding:"required"`       // 套餐名称
	Cover          string  `json:"cover"`                         // 封面图URL
	Category       string  `json:"category"`                      // 分类
	BasePrice      float64 `json:"base_price" binding:"required"` // 基础价格
	DepositRate    float64 `json:"deposit_rate"`                  // 定金比例，如0.3表示30%
	PhotosIncluded int     `json:"photos_included"`               // 包含精修照片数量
	ShootHours     float64 `json:"shoot_hours"`                   // 拍摄时长（小时）
	ContentDesc    string  `json:"content_desc"`                  // 套餐内容描述
	AddonUnitPrice float64 `json:"addon_unit_price"`              // 加选照片单价
	Status         string  `json:"status"`                        // 状态: draft/active/inactive
}

// ======================== 响应 ========================
