package repository

import "testing"

// TestNormalizePage 覆盖分页参数归一化规则：
// 非法页码回退默认值、pageSize 超上限钳制（防全表扫描）。
func TestNormalizePage(t *testing.T) {
	cases := []struct {
		name          string
		page, size    int
		wantPage, want int
	}{
		{"valid", 2, 10, 2, 10},
		{"zero_page", 0, 10, 1, 10},
		{"negative_page", -3, 10, 1, 10},
		{"zero_size", 2, 0, 2, 20},
		{"negative_size", 2, -5, 2, 20},
		{"oversize_capped", 2, 9999, 2, 200},
		{"both_invalid", 0, 0, 1, 20},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			gotPage, gotSize := normalizePage(c.page, c.size)
			if gotPage != c.wantPage || gotSize != c.want {
				t.Errorf("normalizePage(%d,%d) = (%d,%d), want (%d,%d)",
					c.page, c.size, gotPage, gotSize, c.wantPage, c.want)
			}
		})
	}
}
