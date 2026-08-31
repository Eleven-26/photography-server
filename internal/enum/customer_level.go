package enum

// CustomerLevel 客户等级
type CustomerLevel int

const (
	CustomerLevelNormal CustomerLevel = 1 // 普通
	CustomerLevelSilver CustomerLevel = 2 // 银卡
	CustomerLevelGold   CustomerLevel = 3 // 金卡
	CustomerLevelSVIP   CustomerLevel = 4 // SVIP
)

var customerLevelName = map[CustomerLevel]string{
	CustomerLevelNormal: "普通",
	CustomerLevelSilver: "银卡",
	CustomerLevelGold:   "金卡",
	CustomerLevelSVIP:   "SVIP",
}

func CustomerLevelName(level CustomerLevel) string {
	if name, ok := customerLevelName[level]; ok {
		return name
	}
	return "未知"
}
