package service

import (
	"photography-server/internal/repository"
)

type DashboardOverview = repository.DashboardOverview

func (s *Service) Overview(op Operator) (*DashboardOverview, error) {
	return s.DashboardRepo.Overview(op.CompanyID, op.UserID)
}
