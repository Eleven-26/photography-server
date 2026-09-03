package domain

import "time"

// 退款比例档位（按距拍摄开始的小时数）
const (
	RefundRatioFull = 1.0 // >=72h 全额
	RefundRatio80   = 0.8 // >=48h 退80%
	RefundRatio50   = 0.5 // >=24h 退50%
	RefundRatioNone = 0   // <24h 不退
)

// 退款规则档位标识（落库 refund_rule 字段）
const (
	RefundRuleFull = "shoot_gt_72h"
	RefundRule80   = "shoot_48_72h"
	RefundRule50   = "shoot_24_48h"
	RefundRuleNone = "shoot_lt_24h"
)

// RefundRatio 根据距拍摄开始的小时数计算退款比例与规则档位。
// 规则：>=72小时 全额退款；>=48小时 退80%；>=24小时 退50%；<24小时 不退
func RefundRatio(hoursBeforeShoot time.Duration) (ratio float64, rule string) {
	switch {
	case hoursBeforeShoot >= 72*time.Hour:
		return RefundRatioFull, RefundRuleFull
	case hoursBeforeShoot >= 48*time.Hour:
		return RefundRatio80, RefundRule80
	case hoursBeforeShoot >= 24*time.Hour:
		return RefundRatio50, RefundRule50
	default:
		return RefundRatioNone, RefundRuleNone
	}
}
