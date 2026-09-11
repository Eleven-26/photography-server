package model

import (
	"photography-server/internal/enum"
)

// OrderReschedule 订单改期单
type OrderReschedule struct {
	TenantBase
	Code         string                 `gorm:"column:code;size:20;not null;uniqueIndex:uk_reschedule_code,priority:1;comment:改期单编号 RS-xxx" json:"code"`
	OrderID      int64                  `gorm:"column:order_id;index;comment:订单ID" json:"order_id"`
	CustomerID   int64                  `gorm:"column:customer_id;index;comment:客户ID" json:"customer_id"`
	OriginalDate string                 `gorm:"column:original_date;size:20;comment:原拍摄日期" json:"original_date"`
	OriginalTime string                 `gorm:"column:original_time;size:20;comment:原时间段" json:"original_time"`
	NewDate      string                 `gorm:"column:new_date;size:20;comment:新拍摄日期" json:"new_date"`
	NewTime      string                 `gorm:"column:new_time;size:20;comment:新时间段" json:"new_time"`
	FeeType      enum.RescheduleFeeType `gorm:"column:fee_type;type:tinyint;default:1;comment:费用类型 1-免费 2-收调度费 3-不可改期" json:"fee_type"`
	FeeAmount    float64                `gorm:"column:fee_amount;type:decimal(12,2);default:0;comment:调度费金额" json:"fee_amount"`
	ReasonLabel  string                 `gorm:"column:reason_label;size:50;comment:改期原因分类" json:"reason_label"`
	Reason       string                 `gorm:"column:reason;size:500;comment:改期原因说明" json:"reason"`
	Status       enum.RescheduleStatus  `gorm:"column:status;type:tinyint;default:1;comment:状态 1-待确认 2-已同意 3-已拒绝 4-已取消" json:"status"`
	ApplySource  int                    `gorm:"column:apply_source;type:tinyint;default:2;comment:申请来源 1-摄影师发起 2-客户申请" json:"apply_source"`
	AuditBy      int64                  `gorm:"column:audit_by;comment:审批人ID" json:"audit_by"`
	AuditName    string                 `gorm:"column:audit_name;size:50;comment:审批人" json:"audit_name"`
	AuditAt      *string                `gorm:"column:audit_at;comment:审批时间" json:"audit_at"`
	AuditRemark  string                 `gorm:"column:audit_remark;size:200;comment:审批备注" json:"audit_remark"`
}

func (OrderReschedule) TableName() string { return "biz_order_reschedule" }

// OrderReview 订单评价
type OrderReview struct {
	TenantBase
	OrderID      int64   `gorm:"column:order_id;index;comment:订单ID" json:"order_id"`
	CustomerID   int64   `gorm:"column:customer_id;index;comment:客户ID" json:"customer_id"`
	CustomerName string  `gorm:"column:customer_name;size:50;comment:客户姓名(快照)" json:"customer_name"`
	Rating       int     `gorm:"column:rating;type:tinyint;not null;default:5;comment:评分 1-5" json:"rating"`
	Content      string  `gorm:"column:content;size:500;comment:评价内容" json:"content"`
	Images       string  `gorm:"column:images;type:text;comment:评价图片(逗号分隔)" json:"images"`
	IsAnonymous  int     `gorm:"column:is_anonymous;type:tinyint;default:0;comment:是否匿名 0-否 1-是" json:"is_anonymous"`
	Reply        string  `gorm:"column:reply;size:500;comment:摄影师回复" json:"reply"`
	ReplyAt      *string `gorm:"column:reply_at;comment:回复时间" json:"reply_at"`
}

func (OrderReview) TableName() string { return "biz_order_review" }

// CustomRequest 定制需求（客户非套餐需求提交）
type CustomRequest struct {
	TenantBase
	// StoreID 所属门店（0 = 公共池未归属）：独立接单模式下按店过滤；
	// H5 提交页未参数化前新数据落公共池，由员工响应认领
	StoreID      int64                    `gorm:"column:store_id;index;comment:所属门店ID(0=公共池未归属)" json:"store_id"`
	CustomerID   int64                    `gorm:"column:customer_id;index;comment:客户ID(登录提交时有值)" json:"customer_id"`
	Name         string                   `gorm:"column:name;size:50;comment:称呼" json:"name"`
	Mobile       string                   `gorm:"column:mobile;size:20;comment:联系电话" json:"mobile"`
	ProjectType  string                   `gorm:"column:project_type;size:50;comment:拍摄类型" json:"project_type"`
	ExpectedDate string                   `gorm:"column:expected_date;size:50;comment:期望拍摄日期" json:"expected_date"`
	Location     string                   `gorm:"column:location;size:200;comment:期望拍摄地点" json:"location"`
	BudgetMin    float64                  `gorm:"column:budget_min;type:decimal(12,2);comment:预算下限" json:"budget_min"`
	BudgetMax    float64                  `gorm:"column:budget_max;type:decimal(12,2);comment:预算上限" json:"budget_max"`
	Detail       string                   `gorm:"column:detail;size:1000;comment:详细需求" json:"detail"`
	Images       string                   `gorm:"column:images;type:text;comment:参考图片(逗号分隔)" json:"images"`
	Status       enum.CustomRequestStatus `gorm:"column:status;type:tinyint;default:1;comment:状态 1-待处理 2-已响应 3-已关闭" json:"status"`
	LeadID       int64                    `gorm:"column:lead_id;comment:转化线索ID" json:"lead_id"`
	Response     string                   `gorm:"column:response;size:500;comment:响应说明" json:"response"`
	ResponseBy   int64                    `gorm:"column:response_by;comment:响应人ID" json:"response_by"`
	ResponseAt   *string                  `gorm:"column:response_at;comment:响应时间" json:"response_at"`
}

func (CustomRequest) TableName() string { return "biz_custom_request" }

// OrderAddon 订单加项（妆造/加急/收选等）
type OrderAddon struct {
	TenantBase
	OrderID   int64   `gorm:"column:order_id;index;comment:订单ID" json:"order_id"`
	Name      string  `gorm:"column:name;size:100;not null;comment:加项名称" json:"name"`
	Category  string  `gorm:"column:category;size:20;comment:分类 makeup-妆造 urgency-时效 service-服务 retouch-精修" json:"category"`
	Price     float64 `gorm:"column:price;type:decimal(12,2);not null;default:0;comment:单价" json:"price"`
	Qty       int     `gorm:"column:qty;not null;default:1;comment:数量" json:"qty"`
	Amount    float64 `gorm:"column:amount;type:decimal(12,2);not null;default:0;comment:小计(price×qty)" json:"amount"`
	Confirmed int     `gorm:"column:confirmed;type:tinyint;default:0;comment:客户是否确认 0-待确认 1-已确认" json:"confirmed"`
	Remark    string  `gorm:"column:remark;size:200;comment:备注" json:"remark"`
}

func (OrderAddon) TableName() string { return "biz_order_addon" }

// LeadMessage 线索沟通记录（客户咨询/工作室追问/作品分享等往来）
type LeadMessage struct {
	TenantBase
	LeadID     int64  `gorm:"column:lead_id;index;comment:线索ID" json:"lead_id"`
	CustomerID int64  `gorm:"column:customer_id;comment:客户ID" json:"customer_id"`
	Direction  int    `gorm:"column:direction;type:tinyint;not null;default:1;comment:方向 1-客户发来 2-工作室发出" json:"direction"`
	Channel    string `gorm:"column:channel;size:20;comment:渠道 h5-预约页 wechat-微信 sms-短信 phone-电话" json:"channel"`
	Content    string `gorm:"column:content;size:1000;comment:消息内容" json:"content"`
	MsgType    int    `gorm:"column:msg_type;type:tinyint;default:1;comment:类型 1-文本 2-追问 3-报价通知 4-作品分享" json:"msg_type"`
	BizID      int64  `gorm:"column:biz_id;comment:关联业务ID(报价单ID/作品ID等)" json:"biz_id"`
}

func (LeadMessage) TableName() string { return "biz_lead_message" }

// LeadBriefItem 线索 AI 简报项（已确认信息/待追问项）
type LeadBriefItem struct {
	TenantBase
	LeadID         int64                `gorm:"column:lead_id;index;comment:线索ID" json:"lead_id"`
	Title          string               `gorm:"column:title;size:100;comment:信息项标题" json:"title"`
	Value          string               `gorm:"column:value;size:500;comment:已确认的值(已确认项)" json:"value"`
	Question       string               `gorm:"column:question;size:500;comment:待追问问题" json:"question"`
	AiSuggestion   string               `gorm:"column:ai_suggestion;size:500;comment:AI建议追问话术" json:"ai_suggestion"`
	Status         enum.BriefItemStatus `gorm:"column:status;type:tinyint;default:1;comment:状态 1-待追问 2-已发送 3-已确认" json:"status"`
	AffectsPricing int                  `gorm:"column:affects_pricing;type:tinyint;default:0;comment:是否影响报价 0-否 1-是" json:"affects_pricing"`
	Sort           int                  `gorm:"column:sort;default:0;comment:排序" json:"sort"`
	SentAt         *string              `gorm:"column:sent_at;comment:追问发送时间" json:"sent_at"`
	ConfirmedAt    *string              `gorm:"column:confirmed_at;comment:确认时间" json:"confirmed_at"`
}

func (LeadBriefItem) TableName() string { return "biz_lead_brief_item" }

// SlotTemplate 档期时段模板（每周可约规则）
type SlotTemplate struct {
	TenantBase
	StoreID        int64  `gorm:"column:store_id;index;comment:所属门店ID" json:"store_id"`
	PhotographerID int64  `gorm:"column:photographer_id;index;comment:摄影师ID(0=全店通用)" json:"photographer_id"`
	Weekday        int    `gorm:"column:weekday;type:tinyint;not null;default:1;comment:星期几 0-周日 1-周一...6-周六" json:"weekday"`
	StartTime      string `gorm:"column:start_time;size:10;not null;comment:开始时间 HH:mm" json:"start_time"`
	EndTime        string `gorm:"column:end_time;size:10;not null;comment:结束时间 HH:mm" json:"end_time"`
	Status         int    `gorm:"column:status;type:tinyint;not null;default:1;comment:状态 1-启用 0-停用" json:"status"`
}

func (SlotTemplate) TableName() string { return "biz_slot_template" }

// StudioSetting 工作室设置（预约主页/接单规则/改期政策）
type StudioSetting struct {
	Base
	CompanyID           int64   `gorm:"column:company_id;uniqueIndex:uk_studio_setting_company,priority:1;comment:公司ID" json:"company_id"`
	Slogan              string  `gorm:"column:slogan;size:200;comment:宣传语" json:"slogan"`
	Intro               string  `gorm:"column:intro;size:1000;comment:工作室简介" json:"intro"`
	HomepageSlug        string  `gorm:"column:homepage_slug;size:50;comment:预约主页短链标识" json:"homepage_slug"`
	AcceptNew           int     `gorm:"column:accept_new;type:tinyint;not null;default:1;comment:接收新预约 0-暂停 1-接收" json:"accept_new"`
	LockMinutes         int     `gorm:"column:lock_minutes;not null;default:15;comment:下单临时锁定时长(分钟)" json:"lock_minutes"`
	RescheduleFreeHours int     `gorm:"column:reschedule_free_hours;not null;default:72;comment:免费改期剩余小时阈值" json:"reschedule_free_hours"`
	RescheduleFeeRate   float64 `gorm:"column:reschedule_fee_rate;type:decimal(5,2);not null;default:20;comment:改期调度费率(%)" json:"reschedule_fee_rate"`
	RescheduleMinHours  int     `gorm:"column:reschedule_min_hours;not null;default:24;comment:距拍摄不足该小时数不可改期" json:"reschedule_min_hours"`
	SelectDeadlineHours int     `gorm:"column:select_deadline_hours;not null;default:72;comment:上传样片后选片截止(小时)" json:"select_deadline_hours"`
	RetainDays          int     `gorm:"column:retain_days;not null;default:30;comment:未选原片保留天数" json:"retain_days"`
	Faq                 string  `gorm:"column:faq;type:text;comment:常见问题(JSON数组)" json:"faq"`
	ServiceFlow         string  `gorm:"column:service_flow;type:text;comment:服务流程(JSON数组)" json:"service_flow"`
}

func (StudioSetting) TableName() string { return "biz_studio_setting" }

// UserDevice 登录设备
type UserDevice struct {
	TenantBase
	UserID       int64   `gorm:"column:user_id;index;comment:用户ID" json:"user_id"`
	DeviceName   string  `gorm:"column:device_name;size:100;comment:设备名称" json:"device_name"`
	Platform     string  `gorm:"column:platform;size:20;comment:平台 ios/android/pc/wechat" json:"platform"`
	LastIP       string  `gorm:"column:last_ip;size:50;comment:最近登录IP" json:"last_ip"`
	LastActiveAt *string `gorm:"column:last_active_at;comment:最近活跃时间" json:"last_active_at"`
	Status       int     `gorm:"column:status;type:tinyint;not null;default:1;comment:状态 1-正常 0-已踢出" json:"status"`
}

func (UserDevice) TableName() string { return "sys_user_device" }
