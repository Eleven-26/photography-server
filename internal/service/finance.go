package service

import (
	"time"

	"photography-server/internal/model"
	"photography-server/internal/repository"
)

type FinanceSummary = repository.FinanceSummary

func monthRange(month string) (string, string) {
	if len(month) != 7 {
		month = time.Now().Format("2006-01")
	}
	return month + "-01 00:00:00", month + "-31 23:59:59"
}

func (s *Service) FinanceSummary(op Operator, month string) (*FinanceSummary, error) {
	start, end := monthRange(month)
	sum, err := s.FinanceRepo.Summary(op.CompanyID, start, end)
	if err != nil {
		return nil, err
	}
	sum.Month = month
	return sum, nil
}

func (s *Service) ListFinancePayments(op Operator, page, pageSize int, startDate, endDate string) ([]model.OrderPayment, int64, error) {
	return s.FinanceRepo.ListPayments(op.CompanyID, page, pageSize, startDate, endDate)
}

func (s *Service) ListFinanceRefunds(op Operator, page, pageSize int, startDate, endDate string) ([]model.OrderRefund, int64, error) {
	return s.FinanceRepo.ListRefunds(op.CompanyID, page, pageSize, startDate, endDate)
}
