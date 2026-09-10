package service

import (
	"context"
	"time"

	"photography-server/internal/model"
	"photography-server/internal/repository"
)

type FinanceSummary = repository.Summary

// monthRange 返回左闭右开区间 [月初, 下月初)。
// 旧实现用 month+"-31 23:59:59" 拼字符串，对 2 月等不足 31 天的月份会产生
// 非法日期（如 2026-02-31），MySQL 解析报错或退化为 NULL 导致漏统计。
func monthRange(month string) (string, string) {
	t, err := time.ParseInLocation("2006-01", month, time.Local)
	if err != nil {
		now := time.Now()
		t = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)
	}
	start := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.Local)
	end := start.AddDate(0, 1, 0)
	return start.Format("2006-01-02 15:04:05"), end.Format("2006-01-02 15:04:05")
}

func (s *Service) FinanceSummary(ctx context.Context, op Operator, month string) (*repository.Summary, error) {
	start, end := monthRange(month)
	return s.FinanceRepo.GetSummary(ctx, op.CompanyID, start, end)
}

func (s *Service) ListFinancePayments(ctx context.Context, op Operator, page, pageSize int, status string) ([]model.OrderPayment, int64, error) {
	return s.FinanceRepo.ListPayments(ctx, op.CompanyID, page, pageSize, status)
}

// ListFinanceRefunds 退款流水。status 为空表示全部（申请中/已通过/已退款/已驳回）。
func (s *Service) ListFinanceRefunds(ctx context.Context, op Operator, page, pageSize int, status string) ([]model.OrderRefund, int64, error) {
	return s.FinanceRepo.ListRefunds(ctx, op.CompanyID, page, pageSize, status)
}
