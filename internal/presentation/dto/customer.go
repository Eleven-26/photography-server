package dto

// ======================== 请求 ========================

// CustomerCreateReq 创建客户
type CustomerCreateReq struct {
	StoreID  int64  `json:"store_id"`
	Name     string `json:"name" binding:"required"`
	Mobile   string `json:"mobile"`
	Wechat   string `json:"wechat"`
	Gender   string `json:"gender"`
	Birthday string `json:"birthday"`
	Level    string `json:"level"`
	Source   string `json:"source"`
	Tags     string `json:"tags"`
	Status   string `json:"status"`
	Remark   string `json:"remark"`
	Avatar   string `json:"avatar"`
}

// CustomerUpdateReq 更新客户
type CustomerUpdateReq struct {
	StoreID  int64  `json:"store_id"`
	Name     string `json:"name"`
	Mobile   string `json:"mobile"`
	Wechat   string `json:"wechat"`
	Gender   string `json:"gender"`
	Birthday string `json:"birthday"`
	Level    string `json:"level"`
	Source   string `json:"source"`
	Tags     string `json:"tags"`
	Status   string `json:"status"`
	Remark   string `json:"remark"`
	Avatar   string `json:"avatar"`
}

// ======================== 响应 ========================

// CustomerStatsResp 客户统计
type CustomerStatsResp struct {
	Total        int64 `json:"total"`
	Active       int64 `json:"active"`
	GoldUp       int64 `json:"gold_up"`
	NewThisMonth int64 `json:"new_this_month"`
}
