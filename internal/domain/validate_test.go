package domain

import "testing"

func TestIsMobile(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"13800138000", true},
		{"19912345678", true},
		{"  13800138000", false}, // 不做 Trim：调用方负责去除空白
		{"1380013800", false},    // 10 位
		{"138001380000", false},  // 12 位
		{"12800138000", false},   // 第二位为 2
		{"abcdefghijk", false},
		{"+8613800138000", false}, // 带国际区号不支持
		{"", false},
	}
	for _, c := range cases {
		if got := IsMobile(c.in); got != c.want {
			t.Errorf("IsMobile(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}
