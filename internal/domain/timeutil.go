package domain

import "time"

// ParseShootDate 解析拍摄日期为本地时区零点的 time.Time。
// 统一走 time.Local（由启动时 cfg.App.Timezone 加载设置），
// 避免多处各自 time.Parse（UTC）与 time.ParseInLocation(…, time.Local) 混用，
// 导致东八区下"距拍摄小时数"少算 8 小时、退款/改期档位跨档算错（审查报告 #16）。
func ParseShootDate(s string) (time.Time, error) {
	return time.ParseInLocation("2006-01-02", s, time.Local)
}
