package service

import (
	"testing"

	"photography-server/internal/enum"
)

// TestToTinyint 锁定 order 日志状态归一化：
// biz_order_log.from_status / to_status 在 DDL 中是 tinyint，历史上以 interface{} 接收
// int / 枚举并走 fmt.Sprintf("%v", v) 转成 "1" 这类数字串入库，靠 MySQL 隐式转换侥幸通过。
// 现改为显式取整：任何非整型输入都必须落到 0，杜绝向 tinyint 列写字符串触发 ERROR 1366。
func TestToTinyint(t *testing.T) {
	cases := []struct {
		name string
		in   interface{}
		want int
	}{
		{"nil", nil, 0},
		{"int", 7, 7},
		{"int64", int64(6), 6},
		{"uint", uint(3), 3},
		{"order_status_enum", enum.OrderStatusPendingShoot, int(enum.OrderStatusPendingShoot)},
		{"refund_status_enum", enum.RefundStatusApproved, int(enum.RefundStatusApproved)},
		{"string_is_rejected", "图片", 0},
		{"float_is_rejected", 1.5, 0},
		{"bool_is_rejected", true, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := toTinyint(c.in); got != c.want {
				t.Errorf("toTinyint(%#v) = %d, want %d", c.in, got, c.want)
			}
		})
	}
}
