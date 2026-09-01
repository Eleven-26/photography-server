package dto

// ======================== 请求 ========================

// AssetCreateReq 创建作品
type AssetCreateReq struct {
	Title        string `json:"title" binding:"required"`
	Category     string `json:"category"`
	Cover        string `json:"cover"`
	Images       string `json:"images"`
	Description  string `json:"description"`
	Photographer string `json:"photographer"`
	Model        string `json:"model"`
	Location     string `json:"location"`
	Status       string `json:"status"`
}

// AssetUpdateReq 更新作品
type AssetUpdateReq struct {
	Title        string `json:"title" binding:"required"`
	Category     string `json:"category"`
	Cover        string `json:"cover"`
	Images       string `json:"images"`
	Description  string `json:"description"`
	Photographer string `json:"photographer"`
	Model        string `json:"model"`
	Location     string `json:"location"`
	Status       string `json:"status"`
}

// ======================== 响应 ========================
