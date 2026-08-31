package enum

// LogModule 操作日志模块
type LogModule int

const (
	LogModuleCustomer LogModule = 1
	LogModuleLead     LogModule = 2
	LogModuleOrder    LogModule = 3
	LogModulePackage  LogModule = 4
	LogModuleDelivery LogModule = 5
	LogModuleCalendar LogModule = 6
	LogModuleFinance  LogModule = 7
	LogModuleSettings LogModule = 8
)

var logModuleName = map[LogModule]string{
	LogModuleCustomer: "客户",
	LogModuleLead:     "线索",
	LogModuleOrder:    "订单",
	LogModulePackage:  "套餐",
	LogModuleDelivery: "交付",
	LogModuleCalendar: "档期",
	LogModuleFinance:  "财务",
	LogModuleSettings: "设置",
}

func LogModuleName(module LogModule) string {
	if name, ok := logModuleName[module]; ok {
		return name
	}
	return "未知"
}
