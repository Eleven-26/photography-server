package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"photography-server/internal/enum"
	"photography-server/internal/model"
	"photography-server/internal/pkg/errs"
)

// ---------------------------------------------------------------------
// 客户侧报价（报告 H1：C06/C07 查看报价、B3 报价已过期）。
// 复用 PC 端同一套仓储，差别只在归属口径：PC 用 Operator（租户内任意报价可操作），
// 客户只能操作挂在自己名下、或自己线索上的报价。
// ---------------------------------------------------------------------

// ClientQuotes 我的报价单列表（倒序，最新在前）
func (s *Service) ClientQuotes(ctx context.Context, cu *ClientUser) ([]model.Quote, error) {
	return s.LeadRepo.ListQuotesByCustomer(ctx, cu.CompanyID, cu.CustomerID)
}

// ClientQuoteDetail 单张报价详情。
// 与 /quote/list 同一份数据（列表已含明细），独立接口是为了报价详情页可按 id 直取，
// 不必先拉全量列表再前端筛选；归属校验复用 getOwnedQuote（含线索兜底）。
func (s *Service) ClientQuoteDetail(ctx context.Context, cu *ClientUser, quoteID int64) (*model.Quote, error) {
	return s.getOwnedQuote(ctx, cu, quoteID)
}

// getOwnedQuote 取报价单并校验归属当前客户。
// 「不存在」与「无权」返回同一提示，避免用 ID 遍历探测他人报价。
func (s *Service) getOwnedQuote(ctx context.Context, cu *ClientUser, quoteID int64) (*model.Quote, error) {
	q, err := s.LeadRepo.GetQuoteByID(ctx, cu.CompanyID, quoteID)
	if err != nil {
		return nil, errs.NotFound(errs.ErrQuoteNotFound)
	}
	if q.CustomerID > 0 && q.CustomerID == cu.CustomerID {
		return q, nil
	}
	// 早期数据可能未回填 quote.customer_id，回落到线索归属判断
	if q.LeadID > 0 {
		if l, err := s.LeadRepo.GetByID(ctx, cu.CompanyID, q.LeadID); err == nil && l.CustomerID == cu.CustomerID {
			return q, nil
		}
	}
	return nil, errs.NotFound(errs.ErrQuoteNotFound)
}

// ClientQuoteAccept 客户接受报价：置「已接受」并把线索回写为「已成交」，
// 与 PC 端 UpdateQuoteStatus 保持同一口径（两端看到的状态必须一致）。
// 幂等：重复点击直接返回成功。
func (s *Service) ClientQuoteAccept(ctx context.Context, cu *ClientUser, quoteID int64) error {
	q, err := s.getOwnedQuote(ctx, cu, quoteID)
	if err != nil {
		return err
	}
	if q.Status == enum.QuoteStatusAccepted || q.Status == enum.QuoteStatusConverted {
		return nil
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	if err := s.LeadRepo.UpdateQuote(ctx, cu.CompanyID, quoteID, map[string]interface{}{
		"status":     enum.QuoteStatusAccepted,
		"accept_at":  now,
		"updated_by": cu.CustomerID,
	}); err != nil {
		return err
	}
	if q.LeadID > 0 {
		_ = s.LeadRepo.Update(ctx, cu.CompanyID, q.LeadID, map[string]interface{}{
			"status":     enum.LeadStatusConfirmed,
			"updated_by": cu.CustomerID,
		})
	}
	s.NotifyStaff(ctx, clientOperator(cu), q.OwnerID, "order", "客户已接受报价",
		fmt.Sprintf("报价单 %s（%.2f 元）已被客户接受，请安排后续下单", q.Code, q.TotalPrice),
		"quote", q.ID)
	return nil
}

// ClientQuoteModify 客户对报价提出修改意见。
// 刻意不改报价状态：把「希望调整」记成「已拒绝」会让工作室误判客户流失。
// 意见写入线索沟通记录（两端唯一往来载体）并通知负责人，调整后由工作室重新发送报价。
func (s *Service) ClientQuoteModify(ctx context.Context, cu *ClientUser, quoteID int64, content string) error {
	content = strings.TrimSpace(content)
	if content == "" {
		return errs.BadRequest("修改意见不能为空")
	}
	q, err := s.getOwnedQuote(ctx, cu, quoteID)
	if err != nil {
		return err
	}
	if q.LeadID <= 0 {
		return errs.BadRequest("该报价未关联线索，请联系工作室")
	}
	now := time.Now()
	m := model.LeadMessage{
		TenantBase: model.TenantBase{
			Base:      model.Base{CreatedAt: now, UpdatedAt: now, CreatedBy: cu.CustomerID, UpdatedBy: cu.CustomerID},
			CompanyID: cu.CompanyID,
		},
		LeadID:     q.LeadID,
		CustomerID: cu.CustomerID,
		Direction:  1, // 1-客户发来
		Channel:    "h5",
		Content:    content,
		MsgType:    1, // 1-文本
		BizID:      q.ID,
	}
	if err := s.LeadExtraRepo.CreateMessage(ctx, &m); err != nil {
		return err
	}
	_ = s.LeadRepo.Update(ctx, cu.CompanyID, q.LeadID, map[string]interface{}{
		"last_follow_at": now.Format("2006-01-02 15:04:05"),
	})
	s.NotifyStaff(ctx, clientOperator(cu), q.OwnerID, "order", "客户对报价提出修改",
		fmt.Sprintf("客户对报价单 %s 提出修改意见，请在沟通记录中查看并调整", q.Code),
		"quote", q.ID)
	return nil
}
