package dto

// ======================== 请求 ========================

// AssetCreateReq 创建作品
type AssetCreateReq struct {
	Title        string `json:"title" binding:"required"` // 作品标题
	Category     string `json:"category"`                 // 分类
	Cover        string `json:"cover"`                    // 封面图URL
	Images       string `json:"images"`                   // 图片列表，JSON数组
	Description  string `json:"description"`              // 作品描述
	Photographer string `json:"photographer"`             // 摄影师姓名
	Model        string `json:"model"`                    // 模特姓名
	Location     string `json:"location"`                 // 拍摄地点
	Status       string `json:"status"`                   // 状态: draft/published
}

// AssetUpdateReq 更新作品
type AssetUpdateReq struct {
	Title        string `json:"title" binding:"required"` // 作品标题
	Category     string `json:"category"`                 // 分类
	Cover        string `json:"cover"`                    // 封面图URL
	Images       string `json:"images"`                   // 图片列表，JSON数组
	Description  string `json:"description"`              // 作品描述
	Photographer string `json:"photographer"`             // 摄影师姓名
	Model        string `json:"model"`                    // 模特姓名
	Location     string `json:"location"`                 // 拍摄地点
	Status       string `json:"status"`                   // 状态: draft/published
}

// ======================== 响应 ========================
