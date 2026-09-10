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

// DeliveryCreateReq 新建交付任务（PC 交付工作台）。
// OrderID 通常来自路径 :order_id，故不加 binding:required —— 由 service 统一校验。
type DeliveryCreateReq struct {
	OrderID        int64  `json:"order_id"`        // 关联订单ID（路径参数优先）
	Stage          int    `json:"stage"`           // 起始阶段 1-待上传样片 2-客户选片中 3-精修进行中 4-待确认交付（0=默认 1）
	OperatorID     int64  `json:"operator_id"`     // 负责人（员工ID，0=未指派）
	RawCount       int    `json:"raw_count"`       // 原片数量（计划）
	RetouchTarget  int    `json:"retouch_target"`  // 计划精修张数
	SelectDeadline string `json:"select_deadline"` // 选片截止时间（YYYY-MM-DD HH:mm:ss，空=不设）
	Remark         string `json:"remark"`          // 备注
}

// DeliverySelectReq 客户选片
type DeliverySelectReq struct {
	ItemIDs []int64 `json:"item_ids"` // 选中的交付项ID列表
}

// ======================== 响应 ========================
