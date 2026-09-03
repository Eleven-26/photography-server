package domain

import (
	"regexp"
	"testing"
	"time"

	"photography-server/internal/enum"
)

func TestOrderCanTransit(t *testing.T) {
	cases := []struct {
		name string
		from enum.OrderStatus
		to   enum.OrderStatus
		want bool
	}{
		// 正向：主链路推进
		{"deposit->shoot", enum.OrderStatusPendingDeposit, enum.OrderStatusPendingShoot, true},
		{"shoot->shooting", enum.OrderStatusPendingShoot, enum.OrderStatusShooting, true},
		{"shooting->retouching", enum.OrderStatusShooting, enum.OrderStatusRetouching, true},
		{"retouching->delivery", enum.OrderStatusRetouching, enum.OrderStatusPendingDelivery, true},
		{"delivery->completed", enum.OrderStatusPendingDelivery, enum.OrderStatusCompleted, true},
		// 正向：取消
		{"deposit->cancel", enum.OrderStatusPendingDeposit, enum.OrderStatusCancelled, true},
		{"shooting->cancel", enum.OrderStatusShooting, enum.OrderStatusCancelled, true},
		// 反向：跳步/回退不允许
		{"skip-stage", enum.OrderStatusPendingDeposit, enum.OrderStatusCompleted, false},
		{"rollback", enum.OrderStatusShooting, enum.OrderStatusPendingShoot, false},
		{"deposit->retouching", enum.OrderStatusPendingDeposit, enum.OrderStatusRetouching, false},
		// 终态：已完成/已取消不可再流转
		{"completed->any", enum.OrderStatusCompleted, enum.OrderStatusPendingShoot, false},
		{"cancelled->any", enum.OrderStatusCancelled, enum.OrderStatusPendingShoot, false},
		{"completed->cancelled", enum.OrderStatusCompleted, enum.OrderStatusCancelled, false},
		// 自循环不允许
		{"self-loop", enum.OrderStatusPendingShoot, enum.OrderStatusPendingShoot, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := OrderCanTransit(c.from, c.to); got != c.want {
				t.Errorf("OrderCanTransit(%v,%v) = %v, want %v", c.from, c.to, got, c.want)
			}
		})
	}
}

func TestOrderAllowedTransitions(t *testing.T) {
	cases := []struct {
		from enum.OrderStatus
		want []enum.OrderStatus
	}{
		{enum.OrderStatusPendingDeposit, []enum.OrderStatus{enum.OrderStatusPendingShoot, enum.OrderStatusCancelled}},
		{enum.OrderStatusPendingShoot, []enum.OrderStatus{enum.OrderStatusShooting, enum.OrderStatusCancelled}},
		{enum.OrderStatusShooting, []enum.OrderStatus{enum.OrderStatusRetouching, enum.OrderStatusCancelled}},
		{enum.OrderStatusRetouching, []enum.OrderStatus{enum.OrderStatusPendingDelivery, enum.OrderStatusCancelled}},
		{enum.OrderStatusPendingDelivery, []enum.OrderStatus{enum.OrderStatusCompleted, enum.OrderStatusCancelled}},
		{enum.OrderStatusCompleted, []enum.OrderStatus{}},
		{enum.OrderStatusCancelled, []enum.OrderStatus{}},
	}
	for _, c := range cases {
		got := OrderAllowedTransitions(c.from)
		if len(got) != len(c.want) {
			t.Errorf("OrderAllowedTransitions(%v) len = %d, want %d (%v)", c.from, len(got), len(c.want), got)
			continue
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Errorf("OrderAllowedTransitions(%v)[%d] = %v, want %v", c.from, i, got[i], c.want[i])
			}
		}
	}

	// 返回值必须是副本：外部修改不影响状态机
	from := enum.OrderStatusPendingDeposit
	got := OrderAllowedTransitions(from)
	if len(got) > 0 {
		got[0] = enum.OrderStatusCompleted
	}
	again := OrderAllowedTransitions(from)
	if again[0] == enum.OrderStatusCompleted {
		t.Error("OrderAllowedTransitions leaked internal state: mutation affected source map")
	}
}

func TestRefundRatio(t *testing.T) {
	h := 72 * time.Hour
	cases := []struct {
		name     string
		hours    time.Duration
		wantRate float64
		wantRule string
	}{
		{"exactly_72h_full", h, RefundRatioFull, RefundRuleFull},
		{"over_72h_full", 72*time.Hour + time.Minute, RefundRatioFull, RefundRuleFull},
		{"71h59m_80", 72*time.Hour - time.Minute, RefundRatio80, RefundRule80},
		{"exactly_48h_80", 48 * time.Hour, RefundRatio80, RefundRule80},
		{"47h59m_50", 48*time.Hour - time.Minute, RefundRatio50, RefundRule50},
		{"exactly_24h_50", 24 * time.Hour, RefundRatio50, RefundRule50},
		{"23h59m_none", 24*time.Hour - time.Minute, RefundRatioNone, RefundRuleNone},
		{"zero_none", 0, RefundRatioNone, RefundRuleNone},
		{"already_shot_negative_none", -time.Hour, RefundRatioNone, RefundRuleNone},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rate, rule := RefundRatio(c.hours)
			if rate != c.wantRate || rule != c.wantRule {
				t.Errorf("RefundRatio(%v) = (%v,%q), want (%v,%q)", c.hours, rate, rule, c.wantRate, c.wantRule)
			}
		})
	}
}

func TestRound2(t *testing.T) {
	cases := []struct {
		in   float64
		want float64
	}{
		{0, 0},
		{3.14159, 3.14},
		{123.456, 123.46},
		{2.5, 2.5},
		{9.999, 10.0},
		{0.004, 0.0},
		{0.005, 0.01}, // 四舍五入（非银行家舍入）
	}
	for _, c := range cases {
		if got := Round2(c.in); got != c.want {
			t.Errorf("Round2(%v) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestGenCode(t *testing.T) {
	re := regexp.MustCompile(`^SL-\d{6}-\d{4}$`)
	code := GenCode("SL")
	if !re.MatchString(code) {
		t.Errorf("GenCode(\"SL\") = %q, want format SL-YYMMDD-XXXX", code)
	}
	// 前缀必须原样透传
	if code[:3] != "SL-" {
		t.Errorf("GenCode prefix mismatch: %q", code)
	}
	// 同一毫秒内生成可能重复（随机数 0000-9999），此处只验证长度与形态
	// 形态：prefix(2) + '-' + YYMMDD(6) + '-' + 4位随机数 = 14 字符
	if len(code) != 14 {
		t.Errorf("GenCode length = %d, want 14", len(code))
	}
}
