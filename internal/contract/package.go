package contract

import "photography-server/internal/enum"

// ======================== 请求 ========================

// PackageReq 创建/更新套餐
type PackageReq struct {
	StoreID        int64              `json:"store_id"`                      // 门店ID
	Name           string             `json:"name" binding:"required"`       // 套餐名称
	Cover          string             `json:"cover"`                         // 封面图URL
	Category       string             `json:"category"`                      // 分类
	BasePrice      float64            `json:"base_price" binding:"required"` // 基础价格
	DepositRate    float64            `json:"deposit_rate"`                  // 定金比例，**百分数**（30 表示 30%；与 DDL decimal(5,2) DEFAULT 30.00 一致）
	PhotosIncluded int                `json:"photos_included"`               // 包含精修照片数量
	ShootHours     float64            `json:"shoot_hours"`                   // 拍摄时长（小时）
	ContentDesc    string             `json:"content_desc"`                  // 套餐内容描述
	AddonUnitPrice float64            `json:"addon_unit_price"`              // 加选照片单价
	Status         enum.PackageStatus `json:"status"`                        // 状态 1-草稿 2-已上架 3-已下线（0=保持原值）
}

// ======================== 响应 ========================
