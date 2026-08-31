package enum

// LeadStatus 线索状态
type LeadStatus int

const (
	LeadStatusNew       LeadStatus = 1 // 新线索
	LeadStatusFollowing LeadStatus = 2 // 跟进中
	LeadStatusQuoting   LeadStatus = 3 // 报价中
	LeadStatusConverted LeadStatus = 4 // 已转化
	LeadStatusLost      LeadStatus = 5 // 已流失
)

var leadStatusName = map[LeadStatus]string{
	LeadStatusNew:       "新线索",
	LeadStatusFollowing: "跟进中",
	LeadStatusQuoting:   "报价中",
	LeadStatusConverted: "已转化",
	LeadStatusLost:      "已流失",
}

func LeadStatusName(status LeadStatus) string {
	if name, ok := leadStatusName[status]; ok {
		return name
	}
	return "未知"
}
