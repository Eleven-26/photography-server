package enum

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
