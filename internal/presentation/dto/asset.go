package dto

import "photography-server/internal/enum"

// ======================== 请求 ========================

// AssetCreateReq 创建作品
type AssetCreateReq struct {
	Title         string           `json:"title" binding:"required"` // 作品标题
	Category      string           `json:"category"`                 // 分类
	Cover         string           `json:"cover"`                    // 封面图URL
	Images        string           `json:"images"`                   // 图片列表，逗号分隔
	Description   string           `json:"description"`              // 作品描述
	Photographer  string           `json:"photographer"`             // 摄影师姓名
	Model         string           `json:"model"`                    // 模特姓名
	Location      string           `json:"location"`                 // 拍摄地点
	ShootDate     string           `json:"shoot_date"`               // 拍摄日期
	PackageIDs    string           `json:"package_ids"`              // 关联套餐ID，逗号分隔
	Status        enum.AssetStatus `json:"status"`                   // 状态 1-草稿 2-已发布
	Visibility    int              `json:"visibility"`               // 可见性 1-公开 2-未公开（0=默认公开）
	Featured      int              `json:"featured"`                 // 精选展示 0-否 1-是
	Authorization int              `json:"authorization"`            // 客户授权 1-待授权 2-已授权（0=默认已授权）
}

// AssetUpdateReq 更新作品。
// 状态/可见性/精选/授权用指针：featured=0（取消精选）是合法取值，
// 不能用零值表示"未传"，否则取消精选会被当成"保持原值"而静默失效。
type AssetUpdateReq struct {
	Title         string            `json:"title" binding:"required"` // 作品标题
	Category      string            `json:"category"`                 // 分类
	Cover         string            `json:"cover"`                    // 封面图URL
	Images        string            `json:"images"`                   // 图片列表，逗号分隔
	Description   string            `json:"description"`              // 作品描述
	Photographer  string            `json:"photographer"`             // 摄影师姓名
	Model         string            `json:"model"`                    // 模特姓名
	Location      string            `json:"location"`                 // 拍摄地点
	ShootDate     string            `json:"shoot_date"`               // 拍摄日期
	PackageIDs    string            `json:"package_ids"`              // 关联套餐ID，逗号分隔
	Status        *enum.AssetStatus `json:"status"`                   // 状态 1-草稿 2-已发布（nil=保持原值）
	Visibility    *int              `json:"visibility"`               // 可见性 1-公开 2-未公开（nil=保持原值）
	Featured      *int              `json:"featured"`                 // 精选展示 0-否 1-是（nil=保持原值）
	Authorization *int              `json:"authorization"`            // 客户授权 1-待授权 2-已授权（nil=保持原值）
}

// AssetFlagsReq 作品「状态 / 可见性 / 精选」的独立开关。
// 全部使用指针：0 是合法取值（取消精选、切草稿），不能用零值表示"未传"。
type AssetFlagsReq struct {
	Status     *enum.AssetStatus `json:"status"`     // 1-草稿 2-已发布
	Visibility *int              `json:"visibility"` // 1-公开 2-未公开
	Featured   *int              `json:"featured"`   // 0-否 1-是
}

// ======================== 响应 ========================
