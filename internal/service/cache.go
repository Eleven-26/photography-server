package service

import (
	"context"

	"photography-server/internal/pkg/authcache"
)

// 认证画像缓存失效助手（见 internal/pkg/authcache）：
// 员工/客户的状态、角色、密码等发生变更时主动删除对应缓存，使变更即时生效，
// 不必等待 60s TTL 自然过期——"停用账号/重置密码=立即踢下线"是安全语义，不能滞后。
// Redis 不可用（nil）时静默跳过：缓存侧本身 fail-open，下次认证会回源 DB 拿最新画像。

func (s *Service) invalidateStaffCache(ctx context.Context, userID int64) {
	rdb := s.redis()
	if rdb == nil {
		return
	}
	authcache.DelStaff(ctx, rdb, userID)
}

func (s *Service) invalidateCustomerCache(ctx context.Context, companyID, customerID int64) {
	rdb := s.redis()
	if rdb == nil {
		return
	}
	authcache.DelCustomer(ctx, rdb, companyID, customerID)
}
