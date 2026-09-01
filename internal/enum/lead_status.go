package enum

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
