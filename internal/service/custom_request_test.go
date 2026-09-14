package service

import (
	"strings"
	"testing"
	"unicode/utf8"

	"photography-server/internal/model"
)

// TestCustomRequestRemark 锁定「定制需求 → 订单备注」的摘要拼装口径。
//
// 背景：定制需求本身不含套餐与金额结构，转单后原始诉求只保留在备注里。
// 若这里漏拼字段（尤其 detail / 预算），运营在订单上看不到客户到底要什么，
// 只能回定制需求列表翻查 —— 因此把字段集合与空值处理钉死。
func TestCustomRequestRemark(t *testing.T) {
	full := &model.CustomRequest{
		ProjectType:  "家庭纪念",
		ExpectedDate: "2026-10-01",
		Location:     "越秀公园",
		BudgetMin:    3000,
		BudgetMax:    5000,
		Detail:       "三口之家，希望自然光",
	}
	got := customRequestRemark(full)
	for _, want := range []string{"来源：定制需求", "类型：家庭纪念", "期望日期：2026-10-01", "期望地点：越秀公园", "预算：3000-5000", "需求：三口之家，希望自然光"} {
		if !strings.Contains(got, want) {
			t.Errorf("备注缺少片段 %q，实际为 %q", want, got)
		}
	}

	// 全空字段：只留来源标识，不得产生「类型：」这类空片段
	empty := customRequestRemark(&model.CustomRequest{})
	if empty != "来源：定制需求" {
		t.Errorf("空需求备注应仅含来源，实际为 %q", empty)
	}

	// 预算只有下限（H5 末档「¥5,000以上」= max 0）：仍要出现，不能整段省略
	minOnly := customRequestRemark(&model.CustomRequest{BudgetMin: 5000})
	if !strings.Contains(minOnly, "预算：5000-0") {
		t.Errorf("仅下限时预算片段应保留，实际为 %q", minOnly)
	}

	// 超长需求（varchar(500) 列）按 rune 截断，且不得截出半个汉字
	long := &model.CustomRequest{Detail: strings.Repeat("拍", 600)}
	out := customRequestRemark(long)
	if !utf8.ValidString(out) {
		t.Error("截断后出现非法 UTF-8（半个汉字）")
	}
	if n := utf8.RuneCountInString(out); n > orderRemarkMaxRunes {
		t.Errorf("截断后长度 %d 超过上限 %d", n, orderRemarkMaxRunes)
	}
}
