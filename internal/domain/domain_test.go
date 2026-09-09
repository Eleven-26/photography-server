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

// #24：金额按"分"整数拆分，恒有 Deposit + Final == Total（精确相等，无 1 分尾差）
func TestSplitOrderAmounts(t *testing.T) {
	cases := []struct {
		base    float64
		rate    float64
		addon   float64
		deposit float64
		final   float64
		total   float64
	}{
		{1000, 30, 0, 300, 700, 1000},
		{1299, 30, 0, 389.7, 909.3, 1299},    // 1299*0.3=389.7 精确
		{0.1, 50, 0, 0.05, 0.05, 0.1},        // 分以下金额不放大
		{1000, 30, 200, 300, 900, 1200},      // 加选进总额，尾款 = 总额 - 定金
		{1999.99, 30, 1.01, 600, 1401, 2001}, // 1999.99*0.3=599.997 → 600；尾款推导 2001-600=1401
	}
	for _, c := range cases {
		dep, fin, tot := SplitOrderAmounts(c.base, c.rate, c.addon)
		if dep != c.deposit || fin != c.final || tot != c.total {
			t.Errorf("SplitOrderAmounts(%v,%v,%v) = (%v,%v,%v), want (%v,%v,%v)",
				c.base, c.rate, c.addon, dep, fin, tot, c.deposit, c.final, c.total)
		}
		// 恒等式：任意用例下 Deposit+Final 必须精确等于 Total
		if dep+fin != tot {
			t.Errorf("恒等式不成立: deposit(%v)+final(%v)!=total(%v)", dep, fin, tot)
		}
	}
}

// #19：支付状态由金额推导，只有全额收齐/全额退清才变更
func TestDerivePaymentStatus(t *testing.T) {
	// 全额收齐 → 已确认
	if st, ok := DerivePaymentStatus(1000, 0, 1000); !ok || st != enum.PaymentStatusConfirmed {
		t.Errorf("全额收齐应推导为已确认(%v)，got %v ok=%v", enum.PaymentStatusConfirmed, st, ok)
	}
	// 部分收款（定金）→ 不推导，保持原状态
	if _, ok := DerivePaymentStatus(300, 0, 1000); ok {
		t.Error("部分收款不应推导为终态")
	}
	// 全额退清 → 已退款
	if st, ok := DerivePaymentStatus(1000, 1000, 1000); !ok || st != enum.PaymentStatusRefunded {
		t.Errorf("全额退清应推导为已退款(%v)，got %v ok=%v", enum.PaymentStatusRefunded, st, ok)
	}
	// 部分退款 → 不推导
	if _, ok := DerivePaymentStatus(1000, 300, 1000); ok {
		t.Error("部分退款不应推导为终态")
	}
}

func TestGenCode(t *testing.T) {
	// 新格式（#17）：prefix-YYMMDD-8位十六进制随机串（crypto/rand 4字节，碰撞概率 ~1/2^32）
	re := regexp.MustCompile(`^SL-\d{6}-[0-9a-f]{8}$`)
	code := GenCode("SL")
	if !re.MatchString(code) {
		t.Errorf("GenCode(\"SL\") = %q, want format SL-YYMMDD-xxxxxxxx", code)
	}
	// 前缀必须原样透传
	if code[:3] != "SL-" {
		t.Errorf("GenCode prefix mismatch: %q", code)
	}
	// 形态：prefix(2) + '-' + YYMMDD(6) + '-' + 8位随机hex = 18 字符
	if len(code) != 18 {
		t.Errorf("GenCode length = %d, want 18", len(code))
	}
	// 同前缀连续生成不得重复（2^32 空间，连续 1000 次碰撞可忽略，用于回归检查随机位实现）
	seen := map[string]bool{}
	for i := 0; i < 1000; i++ {
		c := GenCode("SL")
		if seen[c] {
			t.Fatalf("GenCode collision: %q", c)
		}
		seen[c] = true
	}
}
