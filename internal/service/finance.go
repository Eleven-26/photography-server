package service

import (
	"bytes"
	"context"
	"encoding/csv"
	"strconv"
	"time"

	"photography-server/internal/enum"
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

// FinanceExport 导出对账 CSV（原型「导出对账」）。
// 内容分三段：汇总四口径 → 收款流水 → 退款单据；返回文件名与内容。
// 头部写入 UTF-8 BOM，Excel 直接双击打开不会把中文显示成乱码。
func (s *Service) FinanceExport(ctx context.Context, op Operator, month string) (string, []byte, error) {
	if _, err := time.ParseInLocation("2006-01", month, time.Local); err != nil {
		month = time.Now().Format("2006-01")
	}
	start, end := monthRange(month)

	sum, err := s.FinanceRepo.GetSummary(ctx, op.CompanyID, start, end)
	if err != nil {
		return "", nil, err
	}
	pays, err := s.FinanceRepo.ExportPayments(ctx, op.CompanyID, start, end)
	if err != nil {
		return "", nil, err
	}
	refunds, err := s.FinanceRepo.ExportRefunds(ctx, op.CompanyID, start, end)
	if err != nil {
		return "", nil, err
	}

	var buf bytes.Buffer
	buf.WriteString("\xEF\xBB\xBF")
	w := csv.NewWriter(&buf)

	_ = w.Write([]string{"对账月份", month})
	_ = w.Write([]string{"统计区间", start + " ~ " + end})
	_ = w.Write(nil)

	_ = w.Write([]string{"【汇总】项目", "数值"})
	_ = w.Write([]string{"订单总额(本月应收)", f2(sum.MonthReceivable)})
	_ = w.Write([]string{"已确认到账(本月已收)", f2(sum.MonthReceived)})
	_ = w.Write([]string{"剩余应收", f2(sum.MonthRemaining)})
	_ = w.Write([]string{"待核验申报金额(不计入已收)", f2(sum.PendingVerifyAmount)})
	_ = w.Write([]string{"待核验申报笔数", strconv.FormatInt(sum.PendingVerifyCount, 10)})
	_ = w.Write([]string{"退款中金额", f2(sum.RefundingAmount)})
	_ = w.Write([]string{"退款中笔数", strconv.FormatInt(sum.RefundingCount, 10)})
	_ = w.Write(nil)

	_ = w.Write([]string{"【收款流水】单号", "订单ID", "客户ID", "类型", "金额", "收款方式", "状态", "收款时间", "操作人", "备注"})
	for _, p := range pays {
		_ = w.Write([]string{
			p.Code, strconv.FormatInt(p.OrderID, 10), strconv.FormatInt(p.CustomerID, 10),
			paymentTypeName(p.Type), f2(p.Amount), p.MethodName,
			enum.PaymentStatusName(p.Status), derefStr(p.PaidAt), p.OperatorName, p.Remark,
		})
	}
	_ = w.Write(nil)

	_ = w.Write([]string{"【退款单据】单号", "订单ID", "客户ID", "金额", "退款原因", "退款规则", "申请来源", "状态", "申请人", "审核时间", "退款时间", "审核备注"})
	for _, rf := range refunds {
		_ = w.Write([]string{
			rf.Code, strconv.FormatInt(rf.OrderID, 10), strconv.FormatInt(rf.CustomerID, 10),
			f2(rf.Amount), rf.Reason, rf.RefundRule, refundSourceName(rf.ApplySource),
			enum.RefundStatusName(rf.Status), rf.ApplyName, derefStr(rf.AuditAt), derefStr(rf.RefundAt), rf.AuditRemark,
		})
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return "", nil, err
	}
	return "财务对账-" + month + ".csv", buf.Bytes(), nil
}

// f2 金额保留两位小数（CSV 里不要出现科学计数法或浮点尾差）
func f2(v float64) string { return strconv.FormatFloat(v, 'f', 2, 64) }

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func paymentTypeName(t string) string {
	switch t {
	case "deposit":
		return "定金"
	case "final":
		return "尾款"
	case "addon":
		return "加选"
	default:
		return t
	}
}

func refundSourceName(src int) string {
	if src == 2 {
		return "客户自助"
	}
	return "管理端"
}
