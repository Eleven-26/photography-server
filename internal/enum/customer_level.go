package enum

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
