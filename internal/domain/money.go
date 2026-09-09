package domain

import (
	"math"

	"photography-server/internal/enum"
)

// Round2 金额四舍五入保留两位小数，避免浮点累计误差影响财务数据
func Round2(v float64) float64 {
	return math.Round(v*100) / 100
}

// yuanToFen 金额（元）转整数分：*100 后四舍五入，消除浮点尾差
func yuanToFen(yuan float64) int64 {
	return int64(math.Round(yuan * 100))
}

// fenToYuan 整数分转金额（元）
func fenToYuan(fen int64) float64 {
	return float64(fen) / 100
}

// SplitOrderAmounts 按"分"整数拆分订单三金额（定金/尾款/总额），保证
// Deposit + Final == Total 精确相等（尾款由总额-定金推导，避免两次独立 Round2 造成 1 分差）。
// basePriceYuan: 基础套餐价(元)；depositRatePercent: 定金比例(%)，如 30 表示 30%；addonYuan: 加选金额(元)。
func SplitOrderAmounts(basePriceYuan, depositRatePercent, addonYuan float64) (deposit, final, total float64) {
	baseFen := yuanToFen(basePriceYuan)
	addonFen := yuanToFen(addonYuan)
	totalFen := baseFen + addonFen

	depositFen := int64(math.Round(float64(baseFen) * depositRatePercent / 100))
	if depositFen < 0 {
		depositFen = 0
	}
	if depositFen > totalFen {
		depositFen = totalFen
	}
	finalFen := totalFen - depositFen
	return fenToYuan(depositFen), fenToYuan(finalFen), fenToYuan(totalFen)
}

// Ge/Le 分精度比较用常量，避免浮点误差
const (
	fenEps     = 0.005 // 0.5 分以内的舍入误差视为相等
	amountZero = 1e-9
)

// FenEps 金额比较允许的舍入误差（元）：判断"累计退款 ≤ 已收"等场景使用
func FenEps() float64 { return fenEps }

// DerivePaymentStatus 由订单金额聚合推导支付状态（替代"收款确认/退款通过即无条件覆写"）：
//   - 已退金额 ≥ 已收金额（且已收 >0） → 已退款（全额退清，含付清后退清）
//   - 全额收齐且未发生任何退款      → 已确认（全额收齐）
//   - 其余（部分收款/部分退款等中间态）→ 返回 ok=false，调用方保持原状态，
//     避免"只付定金也标已确认、只退部分也标已退款"的语义破坏。
//
// 判定顺序注意：付清后退清（paid==refund==total）必须先落入"已退款"，
// 不能先被"全额收齐"截胡（本函数曾在单元测试中暴露该顺序缺陷）。
// 报告 #19：payment_status 应由 paid_amt vs total_amt、refund_amt vs paid_amt 推导。
func DerivePaymentStatus(paidAmt, refundAmt, totalAmt float64) (enum.PaymentStatus, bool) {
	if paidAmt > amountZero && refundAmt+fenEps >= paidAmt {
		return enum.PaymentStatusRefunded, true
	}
	if refundAmt <= amountZero && totalAmt > amountZero && paidAmt+fenEps >= totalAmt {
		return enum.PaymentStatusConfirmed, true
	}
	return enum.PaymentStatusPending, false
}
