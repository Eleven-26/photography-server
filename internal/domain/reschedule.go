package domain

import (
	"time"

	"photography-server/internal/enum"
)

// 改期规则默认值（可被工作室设置覆盖）
const (
	RescheduleFreeHours = 72 // 距拍摄 >=72h 免费改期
	RescheduleMinHours  = 24 // 距拍摄 <24h 不可改期
	RescheduleFeeRate   = 20 // 72h 内改期收取的调度费率（%）
)

// ReschedulePolicy 改期规则（来自工作室设置，缺省用内置默认值）
type ReschedulePolicy struct {
	FreeHours int     // 距拍摄超过该小时数免费改期
	MinHours  int     // 距拍摄不足该小时数不可改期
	FeeRate   float64 // 收费档位的调度费率(%)
}

func (p ReschedulePolicy) normalized() ReschedulePolicy {
	if p.FreeHours <= 0 {
		p.FreeHours = RescheduleFreeHours
	}
	if p.MinHours <= 0 {
		p.MinHours = RescheduleMinHours
	}
	if p.FeeRate <= 0 {
		p.FeeRate = RescheduleFeeRate
	}
	return p
}

// RescheduleFee 按距拍摄开始的小时数计算改期费用。
// 返回：费用类型、调度费金额、规则说明。
// 规则：>= FreeHours 免费；(MinHours, FreeHours) 区间收 FeeRate% 调度费（基数=订单总额）；
// <= MinHours 不可改期。
func RescheduleFee(hoursBeforeShoot time.Duration, totalAmt float64, policy ReschedulePolicy) (enum.RescheduleFeeType, float64, string) {
	p := policy.normalized()
	switch {
	case hoursBeforeShoot < time.Duration(p.MinHours)*time.Hour:
		return enum.RescheduleFeeForbidden, 0, "距拍摄不足24小时，不可改期"
	case hoursBeforeShoot >= time.Duration(p.FreeHours)*time.Hour:
		return enum.RescheduleFeeFree, 0, "免费改期"
	default:
		fee := Round2(totalAmt * p.FeeRate / 100)
		return enum.RescheduleFeeCharged, fee, "改期将收取20%调度费"
	}
}

// ExtraRetouchFee 计算客户加选精修费用：加选张数 × 加选单价。
// extra = selected - included；未超出时为 0。
func ExtraRetouchFee(selected, included int, unitPrice float64) (extraCount int, fee float64) {
	if selected <= included || unitPrice <= 0 {
		return 0, 0
	}
	extraCount = selected - included
	return extraCount, Round2(float64(extraCount) * unitPrice)
}

// FinalBalance 计算确认成片后的最终尾款：套餐尾款 + 加选费用。
func FinalBalance(baseFinalAmt, extraFee float64) float64 {
	return Round2(baseFinalAmt + extraFee)
}

// RefundPreview 取消退款试算结果（供客户取消页与管理端退款审核共用）
type RefundPreview struct {
	Ratio float64 // 退款比例 0-1
	Rule  string  // 规则档位
}

// CalcRefundPreview 按距拍摄时间给出退款比例预览（复用 RefundRatio 规则）
func CalcRefundPreview(hoursBeforeShoot time.Duration) RefundPreview {
	ratio, rule := RefundRatio(hoursBeforeShoot)
	return RefundPreview{Ratio: ratio, Rule: rule}
}
