package service

import (
	"time"

	"photography-server/internal/model"
	"photography-server/internal/repository"
)

type FinanceSummary = repository.Summary

func monthRange(month string) (string, string) {
	if len(month) != 7 {
		month = time.Now().Format("2006-01")
	}
	return month + "-01 00:00:00", month + "-31 23:59:59"
}

func (s *Service) FinanceSummary(op Operator, month string) (*repository.Summary, error) {
	start, end := monthRange(month)
	return s.FinanceRepo.GetSummary(op.CompanyID, start, end)
}

func (s *Service) ListFinancePayments(op Operator, page, pageSize int, status string) ([]model.OrderPayment, int64, error) {
	return s.FinanceRepo.ListPayments(op.CompanyID, page, pageSize, status)
}

func (s *Service) ListFinanceRefunds(op Operator, page, pageSize int) ([]model.OrderRefund, int64, error) {
	return s.FinanceRepo.ListRefunds(op.CompanyID, page, pageSize)
}
