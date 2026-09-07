package domain

import (
	"testing"
	"time"
)

func TestRescheduleFee(t *testing.T) {
	total := 1000.0
	cases := []struct {
		name     string
		hours    time.Duration
		wantType int // enum.RescheduleFeeType 对应值：1免费 2收费 3不可改期
		wantFee  float64
	}{
		{"far_future_free", 100 * time.Hour, 1, 0},
		{"exactly_free_threshold", 72 * time.Hour, 1, 0},
		{"inside_charged_band", 48 * time.Hour, 2, 200}, // 20% × 1000
		{"just_above_min", 24*time.Hour + time.Minute, 2, 200},
		{"exactly_min_charged", 24 * time.Hour, 2, 200},
		{"just_below_min_forbidden", 24*time.Hour - time.Minute, 3, 0},
		{"zero_forbidden", 0, 3, 0},
		{"negative_forbidden", -time.Hour, 3, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			feeType, fee, _ := RescheduleFee(c.hours, total, ReschedulePolicy{})
			if int(feeType) != c.wantType || fee != c.wantFee {
				t.Errorf("RescheduleFee(%v, %v) = (%v, %v), want (%v, %v)",
					c.hours, total, feeType, fee, c.wantType, c.wantFee)
			}
		})
	}
}

func TestRescheduleFeeCustomPolicy(t *testing.T) {
	// 自定义政策覆盖内置默认：免费阈值 48h、最低 12h、费率 10%
	p := ReschedulePolicy{FreeHours: 48, MinHours: 12, FeeRate: 10}
	feeType, fee, _ := RescheduleFee(50*time.Hour, 1000, p)
	if int(feeType) != 1 || fee != 0 {
		t.Errorf("custom policy 50h = (%v, %v), want (1, 0)", feeType, fee)
	}
	feeType, fee, _ = RescheduleFee(20*time.Hour, 1000, p)
	if int(feeType) != 2 || fee != 100 {
		t.Errorf("custom policy 20h = (%v, %v), want (2, 100)", feeType, fee)
	}
	feeType, _, _ = RescheduleFee(10*time.Hour, 1000, p)
	if int(feeType) != 3 {
		t.Errorf("custom policy 10h = %v, want 3(禁止)", feeType)
	}
}

func TestExtraRetouchFee(t *testing.T) {
	cases := []struct {
		name         string
		selected     int
		included     int
		unitPrice    float64
		wantExtraCnt int
		wantFee      float64
	}{
		{"within_included", 30, 30, 50, 0, 0},
		{"below_included", 10, 30, 50, 0, 0},
		{"over_by_5", 35, 30, 50, 5, 250},
		{"zero_unit_price_no_fee", 40, 30, 0, 0, 0}, // 未配置加片单价不加收
		{"fractional_rounding", 31, 30, 33.335, 1, 33.34},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cnt, fee := ExtraRetouchFee(c.selected, c.included, c.unitPrice)
			if cnt != c.wantExtraCnt || fee != c.wantFee {
				t.Errorf("ExtraRetouchFee(%d,%d,%v) = (%d, %v), want (%d, %v)",
					c.selected, c.included, c.unitPrice, cnt, fee, c.wantExtraCnt, c.wantFee)
			}
		})
	}
}

func TestFinalBalance(t *testing.T) {
	if got := FinalBalance(700, 250.005); got != 950.01 {
		t.Errorf("FinalBalance(700, 250.005) = %v, want 950.01", got)
	}
	if got := FinalBalance(0, 0); got != 0 {
		t.Errorf("FinalBalance(0,0) = %v, want 0", got)
	}
}

func TestCalcRefundPreview(t *testing.T) {
	cases := []struct {
		name      string
		hours     time.Duration
		wantRatio float64
	}{
		{"over_72h_full", 80 * time.Hour, RefundRatioFull},
		{"band_80", 60 * time.Hour, RefundRatio80},
		{"band_50", 30 * time.Hour, RefundRatio50},
		{"too_late_none", 2 * time.Hour, RefundRatioNone},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := CalcRefundPreview(c.hours)
			if p.Ratio != c.wantRatio || p.Rule == "" {
				t.Errorf("CalcRefundPreview(%v) = %+v, want ratio %v with non-empty rule", c.hours, p, c.wantRatio)
			}
		})
	}
}
