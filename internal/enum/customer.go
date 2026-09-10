package enum

// 本文件聚合「客户 / 线索 / 报价 / 定制需求」客户域枚举。
// 原拆分为 customer_status.go / customer_level.go / lead_status.go / quote_status.go /
// client_status.go 中的定制需求与简报部分（#43 合并）。

// CustomerStatus 客户状态
type CustomerStatus int

const (
	CustomerStatusPotential CustomerStatus = 1 // 潜在
	CustomerStatusActive    CustomerStatus = 2 // 活跃
	CustomerStatusInactive  CustomerStatus = 3 // 流失/沉睡
)

var customerStatusName = map[CustomerStatus]string{
	CustomerStatusPotential: "潜在",
	CustomerStatusActive:    "活跃",
	CustomerStatusInactive:  "流失/沉睡",
}

func CustomerStatusName(status CustomerStatus) string {
	if name, ok := customerStatusName[status]; ok {
		return name
	}
	return "未知"
}

// CustomerLevel 客户等级
type CustomerLevel int

const (
	CustomerLevelNormal   CustomerLevel = 1 // 普通
	CustomerLevelGold     CustomerLevel = 2 // 黄金
	CustomerLevelPlatinum CustomerLevel = 3 // 铂金
	CustomerLevelDiamond  CustomerLevel = 4 // 钻石
)

var customerLevelName = map[CustomerLevel]string{
	CustomerLevelNormal:   "普通",
	CustomerLevelGold:     "黄金",
	CustomerLevelPlatinum: "铂金",
	CustomerLevelDiamond:  "钻石",
}

func CustomerLevelName(level CustomerLevel) string {
	if name, ok := customerLevelName[level]; ok {
		return name
	}
	return "未知"
}

// LeadStatus 线索状态
type LeadStatus int

const (
	LeadStatusPending   LeadStatus = 1 // 待回复
	LeadStatusQuoting   LeadStatus = 2 // 待报价
	LeadStatusQuoted    LeadStatus = 3 // 已报价
	LeadStatusConfirmed LeadStatus = 4 // 待确认/已成交
	LeadStatusLose      LeadStatus = 5 // 已流失
)

var leadStatusName = map[LeadStatus]string{
	LeadStatusPending:   "待回复",
	LeadStatusQuoting:   "待报价",
	LeadStatusQuoted:    "已报价",
	LeadStatusConfirmed: "已成交",
	LeadStatusLose:      "已流失",
}

func LeadStatusName(status LeadStatus) string {
	if name, ok := leadStatusName[status]; ok {
		return name
	}
	return "未知"
}

// QuoteStatus 报价单状态
type QuoteStatus int

const (
	QuoteStatusDraft     QuoteStatus = 1 // 草稿
	QuoteStatusSent      QuoteStatus = 2 // 已发送
	QuoteStatusAccepted  QuoteStatus = 3 // 已接受
	QuoteStatusRejected  QuoteStatus = 4 // 已拒绝
	QuoteStatusConverted QuoteStatus = 5 // 已成交
)

var quoteStatusName = map[QuoteStatus]string{
	QuoteStatusDraft:     "草稿",
	QuoteStatusSent:      "已发送",
	QuoteStatusAccepted:  "已接受",
	QuoteStatusRejected:  "已拒绝",
	QuoteStatusConverted: "已成交",
}

func QuoteStatusName(status QuoteStatus) string {
	if name, ok := quoteStatusName[status]; ok {
		return name
	}
	return "未知"
}

// CustomRequestStatus 定制需求状态
type CustomRequestStatus int

const (
	CustomRequestPending   CustomRequestStatus = 1 // 待处理
	CustomRequestResponded CustomRequestStatus = 2 // 已响应
	CustomRequestClosed    CustomRequestStatus = 3 // 已关闭
)

// BriefItemStatus AI 简报项状态
type BriefItemStatus int

const (
	BriefItemPending   BriefItemStatus = 1 // 待追问
	BriefItemSent      BriefItemStatus = 2 // 已发送
	BriefItemConfirmed BriefItemStatus = 3 // 已确认
)
