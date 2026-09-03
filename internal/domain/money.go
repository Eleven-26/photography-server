package domain

import "math"

// Round2 金额四舍五入保留两位小数，避免浮点累计误差影响财务数据
func Round2(v float64) float64 {
	return math.Round(v*100) / 100
}
