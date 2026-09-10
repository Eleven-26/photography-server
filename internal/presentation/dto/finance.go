package dto

// FinanceSummaryResp 财务汇总（PC 财务页）。
//
// 口径说明（原型要求）：待核验申报**不计入已收**，但**计入剩余应收**，
// 因此「剩余应收 = 订单总额 - 已确认到账」可能包含尚未核验的部分。
type FinanceSummaryResp struct {
	// —— 前端口径 ——
	MonthReceivable     float64 `json:"month_receivable"`      // 本月应收（本月成交订单总额）
	MonthReceived       float64 `json:"month_received"`        // 本月已确认到账
	MonthRemaining      float64 `json:"month_remaining"`       // 本月剩余应收
	PendingVerifyCount  int64   `json:"pending_verify_count"`  // 待核验申报笔数
	PendingVerifyAmount float64 `json:"pending_verify_amount"` // 待核验申报金额
	RefundingCount      int64   `json:"refunding_count"`       // 退款中笔数
	RefundingAmount     float64 `json:"refunding_amount"`      // 退款中金额

	// —— 扩展口径（保留）——
	TotalIncome  float64 `json:"total_income"`  // 区间内已确认到账（同 month_received）
	DepositTotal float64 `json:"deposit_total"` // 区间内定金到账
	FinalTotal   float64 `json:"final_total"`   // 区间内尾款+加片到账
	RefundTotal  float64 `json:"refund_total"`  // 区间内已退款金额
	PendingCount int64   `json:"pending_count"` // 待核验笔数（同 pending_verify_count）
}
