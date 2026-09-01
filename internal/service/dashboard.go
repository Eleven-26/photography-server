package service

import "photography-server/internal/repository"

func (s *Service) Overview(op Operator) (*repository.Overview, error) {
	return s.DashboardRepo.GetOverview(op.CompanyID, op.UserID)
}
