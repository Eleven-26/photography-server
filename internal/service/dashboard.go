package service

import (
	"context"

	"photography-server/internal/repository"
)

// Overview 工作台概览。ctx 透传，聚合统计的每条 SQL 都挂到当前请求链路。
func (s *Service) Overview(ctx context.Context, op Operator) (*repository.Overview, error) {
	return s.DashboardRepo.GetOverview(ctx, op.CompanyID, op.UserID)
}
