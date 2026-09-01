package dto

// ======================== 请求 ========================

// DeliveryItemReq 交付项
type DeliveryItemReq struct {
	URL      string `json:"url" binding:"required"`
	FileType string `json:"file_type"`
	Kind     string `json:"kind"`
	Filename string `json:"filename"`
	Size     int64  `json:"size"`
}

// DeliverySelectReq 客户选片
type DeliverySelectReq struct {
	ItemIDs []int64 `json:"item_ids"`
}

// ======================== 响应 ========================
