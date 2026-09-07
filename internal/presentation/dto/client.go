package dto

// ======================== 客户端（H5/小程序） ========================

// ClientSlot 可约时段
type ClientSlot struct {
	StartTime string `json:"start_time"` // HH:mm
	EndTime   string `json:"end_time"`   // HH:mm
	Available bool   `json:"available"`  // 是否可约
}

// ClientBookingReq 客户提交预约
type ClientBookingReq struct {
	PackageID    int64  `json:"package_id" binding:"required"` // 套餐ID
	ShootDate    string `json:"shoot_date" binding:"required"` // 拍摄日期 2006-01-02
	ShootTime    string `json:"shoot_time" binding:"required"` // 拍摄时段 09:00-11:00
	ShootAddress string `json:"shoot_address"`                 // 拍摄地点
	PeopleCount  string `json:"people_count"`                  // 拍摄人数（如 2大1小）
	ShootStyle   string `json:"shoot_style"`                   // 拍摄风格
	Remark       string `json:"remark"`                        // 备注
}

// ClientOrderDetail 客户端订单详情
type ClientOrderDetail struct {
	Order       interface{} `json:"order"`       // 订单信息
	Payments    interface{} `json:"payments"`    // 收款记录
	Refunds     interface{} `json:"refunds"`     // 退款记录
	Logs        interface{} `json:"logs"`        // 操作日志
	Delivery    interface{} `json:"delivery"`    // 交付信息
	Reschedules interface{} `json:"reschedules"` // 改期单
	Addons      interface{} `json:"addons"`      // 加项
	Review      interface{} `json:"review"`      // 评价
}

// ClientRescheduleReq 客户改期申请
type ClientRescheduleReq struct {
	NewDate     string `json:"new_date" binding:"required"` // 新拍摄日期
	NewTime     string `json:"new_time" binding:"required"` // 新拍摄时段
	ReasonLabel string `json:"reason_label"`                // 改期原因分类
	Reason      string `json:"reason"`                      // 改期原因说明
}

// ClientRefundReq 客户退款申请
type ClientRefundReq struct {
	ReasonLabel string `json:"reason_label"` // 取消原因分类
	Reason      string `json:"reason"`       // 退款原因
}

// ClientReviewReq 客户评价
type ClientReviewReq struct {
	Rating      int    `json:"rating" binding:"required"` // 评分 1-5
	Content     string `json:"content"`                   // 评价内容
	Images      string `json:"images"`                    // 评价图片(逗号分隔)
	IsAnonymous int    `json:"is_anonymous"`              // 是否匿名 0-否 1-是
}

// ClientFeedbackReq 客户精修反馈
type ClientFeedbackReq struct {
	Content  string `json:"content"`  // 修改意见
	Types    string `json:"types"`    // 修改类型(逗号分隔)
	Priority string `json:"priority"` // normal-普通 important-重要 urgent-紧急
}

// ClientCustomRequestReq 定制需求提交
type ClientCustomRequestReq struct {
	Name         string  `json:"name"`          // 称呼（游客提交时必填）
	Mobile       string  `json:"mobile"`        // 联系电话（游客提交时必填）
	ProjectType  string  `json:"project_type"`  // 拍摄类型
	ExpectedDate string  `json:"expected_date"` // 期望拍摄日期
	Location     string  `json:"location"`      // 期望拍摄地点
	BudgetMin    float64 `json:"budget_min"`    // 预算下限
	BudgetMax    float64 `json:"budget_max"`    // 预算上限
	Detail       string  `json:"detail"`        // 详细需求
	Images       string  `json:"images"`        // 参考图片(逗号分隔)
}

// ClientSmsCodeReq 客户端发送验证码
type ClientSmsCodeReq struct {
	Mobile string `json:"mobile" binding:"required"` // 手机号
}

// ClientSmsLoginReq 客户端验证码登录（H5/小程序共用）
type ClientSmsLoginReq struct {
	CompanyID int64  `json:"company_id"`                // 工作室ID（多租户路由定位）
	Mobile    string `json:"mobile" binding:"required"` // 手机号
	OpenID    string `json:"openid"`                    // 小程序openid（选填）
	Code      string `json:"code" binding:"required"`   // 验证码
}

// ======================== 摄影师 App ========================

// AppOverview App 工作台待办统计
type AppOverview struct {
	PendingConfirm       int64 `json:"pending_confirm"`        // 待确认预约
	PendingDeposit       int64 `json:"pending_deposit"`        // 待收定金
	PendingShoot         int64 `json:"pending_shoot"`          // 待拍摄
	TodayShoot           int   `json:"today_shoot"`            // 今日拍摄
	PendingDelivery      int64 `json:"pending_delivery"`       // 待交付
	PendingReschedule    int64 `json:"pending_reschedule"`     // 待审批改期
	PendingRefund        int64 `json:"pending_refund"`         // 待审批退款
	PendingCustomRequest int64 `json:"pending_custom_request"` // 待处理定制需求
}

// AppLeadMessageReq 发送线索沟通消息
type AppLeadMessageReq struct {
	Content string `json:"content" binding:"required"` // 消息内容
	Channel string `json:"channel"`                    // 渠道 h5/wechat/sms/phone
	MsgType int64  `json:"msg_type"`                   // 类型 1-文本 2-追问 3-报价通知 4-作品分享
	BizID   int64  `json:"biz_id"`                     // 关联业务ID
}

// AppSlotTemplateReq 档期时段模板
type AppSlotTemplateReq struct {
	PhotographerID int64  `json:"photographer_id"` // 摄影师ID（0=全店通用）
	Weekday        int    `json:"weekday"`         // 星期几 0-周日 ... 6-周六
	StartTime      string `json:"start_time"`      // HH:mm
	EndTime        string `json:"end_time"`        // HH:mm
	Status         int64  `json:"status"`          // 1-启用 0-停用
}

// AppStudioSettingReq 工作室设置更新
type AppStudioSettingReq struct {
	Slogan              string   `json:"slogan"`                // 宣传语
	Intro               string   `json:"intro"`                 // 简介
	HomepageSlug        string   `json:"homepage_slug"`         // 预约主页短链标识
	AcceptNew           *int     `json:"accept_new"`            // 接收新预约 0-暂停 1-接收
	LockMinutes         *int     `json:"lock_minutes"`          // 下单临时锁定时长
	RescheduleFreeHours *int     `json:"reschedule_free_hours"` // 免费改期阈值
	RescheduleFeeRate   *float64 `json:"reschedule_fee_rate"`   // 改期调度费率
	RescheduleMinHours  *int     `json:"reschedule_min_hours"`  // 不可改期阈值
	SelectDeadlineHours *int     `json:"select_deadline_hours"` // 选片截止小时
	RetainDays          *int     `json:"retain_days"`           // 未选原片保留天数
	Faq                 string   `json:"faq"`                   // 常见问题(JSON)
	ServiceFlow         string   `json:"service_flow"`          // 服务流程(JSON)
}

// ToUpdates 组装更新字段：指针非 nil 才更新（支持把数值改为 0）
func (req AppStudioSettingReq) ToUpdates() map[string]interface{} {
	updates := map[string]interface{}{}
	if req.Slogan != "" {
		updates["slogan"] = req.Slogan
	}
	if req.Intro != "" {
		updates["intro"] = req.Intro
	}
	if req.HomepageSlug != "" {
		updates["homepage_slug"] = req.HomepageSlug
	}
	if req.AcceptNew != nil {
		updates["accept_new"] = *req.AcceptNew
	}
	if req.LockMinutes != nil {
		updates["lock_minutes"] = *req.LockMinutes
	}
	if req.RescheduleFreeHours != nil {
		updates["reschedule_free_hours"] = *req.RescheduleFreeHours
	}
	if req.RescheduleFeeRate != nil {
		updates["reschedule_fee_rate"] = *req.RescheduleFeeRate
	}
	if req.RescheduleMinHours != nil {
		updates["reschedule_min_hours"] = *req.RescheduleMinHours
	}
	if req.SelectDeadlineHours != nil {
		updates["select_deadline_hours"] = *req.SelectDeadlineHours
	}
	if req.RetainDays != nil {
		updates["retain_days"] = *req.RetainDays
	}
	if req.Faq != "" {
		updates["faq"] = req.Faq
	}
	if req.ServiceFlow != "" {
		updates["service_flow"] = req.ServiceFlow
	}
	return updates
}

// AppRescheduleAuditReq 改期审批
type AppRescheduleAuditReq struct {
	Approved bool   `json:"approved"` // 是否同意
	Remark   string `json:"remark"`   // 审批备注
}

// AppBriefConfirmReq 简报项确认
type AppBriefConfirmReq struct {
	Value string `json:"value" binding:"required"` // 确认内容
}

// AppReviewReplyReq 评价回复
type AppReviewReplyReq struct {
	Reply string `json:"reply" binding:"required"` // 回复内容
}

// AppCustomRequestRespondReq 定制需求响应
type AppCustomRequestRespondReq struct {
	Response string `json:"response"` // 响应说明
}
