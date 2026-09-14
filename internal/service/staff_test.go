package service

import (
	"strings"
	"testing"
)

// TestNormalizeHomepageSlug 覆盖主页标识的格式收敛与拒绝分支。
// 该标识会被拼进对外分享链接（?slug=），并在 H5 端作为租户定位键，故格式必须收敛。
func TestNormalizeHomepageSlug(t *testing.T) {
	long50 := strings.Repeat("a", 50)
	long51 := strings.Repeat("a", 51)

	cases := []struct {
		name string
		in   string
		want string
		ok   bool
	}{
		{"原样保留", "lu-studio", "lu-studio", true},
		{"去空白并转小写", "  Lu-Studio  ", "lu-studio", true},
		{"允许数字", "studio2026", "studio2026", true},
		{"中间连字符允许", "lu-studio-2", "lu-studio-2", true},
		{"空串放行（视同不修改）", "   ", "", true},
		{"首字符为连字符拒绝", "-lu", "", false},
		{"仅连字符拒绝", "--", "", false},
		{"含下划线拒绝", "lu_studio", "", false},
		{"含中文拒绝", "路先生", "", false},
		{"含空格拒绝", "lu studio", "", false},
		{"路径穿越拒绝", "../etc", "", false},
		{"边界 50 位通过", long50, long50, true},
		{"超 1 位拒绝", long51, "", false},
	}

	for _, c := range cases {
		got, err := NormalizeHomepageSlug(c.in)
		if !c.ok {
			if err == nil {
				t.Errorf("%s: 期望报错，实际通过并返回 %q", c.name, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("%s: 期望通过，实际报错 %v", c.name, err)
			continue
		}
		if got != c.want {
			t.Errorf("%s: 期望 %q，实际 %q", c.name, c.want, got)
		}
	}
}

// TestDefaultHomepageSlug 兜底标识按公司 ID 派生：须稳定、可预期。
func TestDefaultHomepageSlug(t *testing.T) {
	if got := DefaultHomepageSlug(12); got != "studio-12" {
		t.Fatalf("期望 studio-12，实际 %s", got)
	}
	if a, b := DefaultHomepageSlug(7), DefaultHomepageSlug(7); a != b {
		t.Fatalf("同一公司 ID 两次派生结果不一致: %s / %s", a, b)
	}
	// 派生值必须自身满足标识格式约束（否则会写进库、拼进链接后无法通过校验）
	if _, err := NormalizeHomepageSlug(DefaultHomepageSlug(1024)); err != nil {
		t.Fatalf("派生标识未通过格式校验: %v", err)
	}
}
