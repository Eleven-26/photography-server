package dto

// ======================== 请求 ========================

// DeliveryItemReq 交付项
type DeliveryItemReq struct {
	URL      string `json:"url" binding:"required"` // 文件URL
	FileType string `json:"file_type"`              // 文件类型: image/video
	Kind     string `json:"kind"`                   // 种类: sample/original/retouched
	Filename string `json:"filename"`               // 文件名
	Size     int64  `json:"size"`                   // 文件大小（字节）
}

// DeliverySelectReq 客户选片
type DeliverySelectReq struct {
	ItemIDs []int64 `json:"item_ids"` // 选中的交付项ID列表
}

// ======================== 响应 ========================
