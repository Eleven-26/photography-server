package service

import (
	"context"
	"testing"
)

// TestFindOrCreateCustomerByMobile_SkipWhenMobileEmpty 锁定「无手机号不建档」这条约定。
//
// crm_customer.mobile 只有普通索引、没有唯一约束，去重完全依赖本函数；
// 一旦空手机号也建档，同一人多条空号记录会静默堆积，且下次按手机号查档永远命中不到他们。
// 这里刻意传 nil repo：函数必须在触达仓储之前就返回，任何对 repo 的调用都会 panic。
func TestFindOrCreateCustomerByMobile_SkipWhenMobileEmpty(t *testing.T) {
	s := &Service{}
	for _, mobile := range []string{"", "   ", "\t\n"} {
		c, err := s.findOrCreateCustomerByMobile(context.Background(), nil, 1, "张三", mobile, CustomerSourceLead)
		if err != nil {
			t.Errorf("mobile=%q 应静默跳过，却返回错误：%v", mobile, err)
		}
		if c != nil {
			t.Errorf("mobile=%q 不应建档，却返回客户 %+v", mobile, c)
		}
	}
}
