package dto

import "photography-server/internal/enum"

// ======================== 请求 ========================

// CustomerCreateReq 创建客户
// level/status 与 model.Customer 一致，均为 int 枚举（前端按 number 提交）。
type CustomerCreateReq struct {
	StoreID  int64               `json:"store_id"`                // 门店ID
	Name     string              `json:"name" binding:"required"` // 客户姓名
	Mobile   string              `json:"mobile"`                  // 手机号
	Wechat   string              `json:"wechat"`                  // 微信号
	Gender   string              `json:"gender"`                  // 性别: male/female/unknown
	Birthday string              `json:"birthday"`                // 生日
	Level    enum.CustomerLevel  `json:"level"`                   // 客户等级 1-普通 2-黄金 3-铂金 4-钻石
	Source   string              `json:"source"`                  // 来源
	Tags     string              `json:"tags"`                    // 标签，逗号分隔
	Status   enum.CustomerStatus `json:"status"`                  // 状态 1-潜在 2-活跃 3-流失
	Remark   string              `json:"remark"`                  // 备注
	Avatar   string              `json:"avatar"`                  // 头像URL
	// AllowNotifications 用指针：创建时不传默认允许(1)，显式传 0 表示不允许。
	AllowNotifications *int   `json:"allow_notifications"` // 通知许可 0-不允许 1-允许
	PreferStyle        string `json:"prefer_style"`        // 偏好风格
	PreferScene        string `json:"prefer_scene"`        // 常用场景
}

// CustomerUpdateReq 更新客户
// level/status 为 int 枚举，传 0 表示未指定、保持库中原值。
type CustomerUpdateReq struct {
	StoreID  int64               `json:"store_id"` // 门店ID
	Name     string              `json:"name"`     // 客户姓名
	Mobile   string              `json:"mobile"`   // 手机号
	Wechat   string              `json:"wechat"`   // 微信号
	Gender   string              `json:"gender"`   // 性别: male/female/unknown
	Birthday string              `json:"birthday"` // 生日
	Level    enum.CustomerLevel  `json:"level"`    // 客户等级 1-普通 2-黄金 3-铂金 4-钻石
	Source   string              `json:"source"`   // 来源
	Tags     string              `json:"tags"`     // 标签，逗号分隔
	Status   enum.CustomerStatus `json:"status"`   // 状态 1-潜在 2-活跃 3-流失
	Remark   string              `json:"remark"`   // 备注
	Avatar   string              `json:"avatar"`   // 头像URL
	// AllowNotifications 用指针：nil 表示未传（保持原值），显式 0 表示关闭通知。
	// 若用值类型，「关闭通知」会被零值语义吞掉而静默回退（同 Asset.featured 的历史缺陷）。
	AllowNotifications *int   `json:"allow_notifications"` // 通知许可 0-不允许 1-允许
	PreferStyle        string `json:"prefer_style"`        // 偏好风格
	PreferScene        string `json:"prefer_scene"`        // 常用场景
}

// ======================== 响应 ========================

// CustomerStatsResp 客户统计（字段与 PC 前端 customerStats 对齐：
// total/potential/active/inactive 为状态分布口径，gold_up/new_this_month 为扩展口径）
type CustomerStatsResp struct {
	Total        int64 `json:"total"`          // 总数
	Potential    int64 `json:"potential"`      // 潜在客户数（status=1）
	Active       int64 `json:"active"`         // 活跃数（status=2）
	Inactive     int64 `json:"inactive"`       // 非活跃数（status=3）
	GoldUp       int64 `json:"gold_up"`        // 黄金及以上等级数
	NewThisMonth int64 `json:"new_this_month"` // 本月新增

	// —— 复购口径（原型「复购客户 / 复购率」）——
	RepurchaseCount int64   `json:"repurchase_count"` // 复购客户数（下单 ≥ 2 次）
	RepurchaseRate  float64 `json:"repurchase_rate"`  // 复购率 %（分母为有过下单的客户）
}
