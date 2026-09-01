package dto

// ======================== 请求 ========================

// CustomerCreateReq 创建客户
type CustomerCreateReq struct {
	StoreID  int64  `json:"store_id"`                // 门店ID
	Name     string `json:"name" binding:"required"` // 客户姓名
	Mobile   string `json:"mobile"`                  // 手机号
	Wechat   string `json:"wechat"`                  // 微信号
	Gender   string `json:"gender"`                  // 性别: male/female
	Birthday string `json:"birthday"`                // 生日
	Level    string `json:"level"`                   // 等级: normal/gold/vip
	Source   string `json:"source"`                  // 来源
	Tags     string `json:"tags"`                    // 标签，逗号分隔
	Status   string `json:"status"`                  // 状态: active/inactive
	Remark   string `json:"remark"`                  // 备注
	Avatar   string `json:"avatar"`                  // 头像URL
}

// CustomerUpdateReq 更新客户
type CustomerUpdateReq struct {
	StoreID  int64  `json:"store_id"` // 门店ID
	Name     string `json:"name"`     // 客户姓名
	Mobile   string `json:"mobile"`   // 手机号
	Wechat   string `json:"wechat"`   // 微信号
	Gender   string `json:"gender"`   // 性别: male/female
	Birthday string `json:"birthday"` // 生日
	Level    string `json:"level"`    // 等级: normal/gold/vip
	Source   string `json:"source"`   // 来源
	Tags     string `json:"tags"`     // 标签，逗号分隔
	Status   string `json:"status"`   // 状态: active/inactive
	Remark   string `json:"remark"`   // 备注
	Avatar   string `json:"avatar"`   // 头像URL
}

// ======================== 响应 ========================

// CustomerStatsResp 客户统计
type CustomerStatsResp struct {
	Total        int64 `json:"total"`          // 总数
	Active       int64 `json:"active"`         // 活跃数
	GoldUp       int64 `json:"gold_up"`        // 升级金牌数
	NewThisMonth int64 `json:"new_this_month"` // 本月新增
}
