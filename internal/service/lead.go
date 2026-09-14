package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"photography-server/internal/domain"
	"photography-server/internal/enum"
	"photography-server/internal/model"
	"photography-server/internal/pkg/errs"
	"photography-server/internal/presentation/dto"
	"photography-server/internal/repository"
)

func (s *Service) ListLeads(ctx context.Context, op Operator, page, pageSize int, keyword, status string, ownerID int64) ([]model.Lead, int64, error) {
	return s.LeadRepo.List(ctx, op.CompanyID, page, pageSize, keyword, status, ownerID)
}

func (s *Service) GetLeadDetail(ctx context.Context, op Operator, id int64) (*model.Lead, error) {
	l, err := s.LeadRepo.GetByID(ctx, op.CompanyID, id)
	if err != nil {
		return nil, errs.NotFound(errs.ErrLeadNotFound)
	}
	return l, nil
}

func (s *Service) CreateLead(ctx context.Context, op Operator, req dto.LeadCreateReq) (*model.Lead, error) {
	l := model.Lead{
		TenantBase: model.TenantBase{
			Base:      model.Base{CreatedBy: op.UserID, UpdatedBy: op.UserID},
			CompanyID: op.CompanyID,
		},
		Code:        domain.GenCode("LD"),
		StoreID:     op.StoreID,
		Name:        req.Name,
		Mobile:      req.Mobile,
		Source:      req.Source,
		ProjectType: req.ProjectType,
		BudgetMin:   req.BudgetMin,
		BudgetMax:   req.BudgetMax,
		ShootDate:   strPtr(req.ShootDate),
		Remark:      req.Remark,
		OwnerID:     orDefaultInt64(req.OwnerID, op.UserID),
		Status:      enum.LeadStatusPending,
	}
	// 建档与建线索放在同一事务：按手机号在客户表查档，没有则以线索姓名/来源建档。
	// 手机号为空时不建档，CustomerID 保持 0（无手机号无法去重），
	// 待「转客户 / 转订单」时由用户补全手机号再建。
	err := repository.Tx(func(tx *gorm.DB) error {
		c, cerr := s.findOrCreateCustomerByMobile(ctx, s.CustomerRepo.WithTx(tx),
			op.CompanyID, req.Name, req.Mobile, orDefault(req.Source, CustomerSourceLead))
		if cerr != nil {
			return cerr
		}
		if c != nil {
			l.CustomerID = c.ID
		}
		return s.LeadRepo.WithTx(tx).Create(ctx, &l)
	})
	if err != nil {
		return nil, err
	}
	return &l, nil
}

// UpdateLead 局部更新：未传字段保持库中原值。
// 前端「标记流失」「改状态」只会传 status，若按全量覆盖会把姓名/手机号/备注清空。
func (s *Service) UpdateLead(ctx context.Context, op Operator, id int64, req dto.LeadUpdateReq) error {
	l, err := s.LeadRepo.GetByID(ctx, op.CompanyID, id)
	if err != nil {
		return errs.NotFound(errs.ErrLeadNotFound)
	}
	// 变更归属人 = 分配线索，属高敏动作（决定谁跟进这个客户）。路由级 lead/update
	// 无法区分字段，故在此按"负责人是否真的变了"追加 lead:assign 校验；同人不变更放行。
	if req.OwnerID != 0 && req.OwnerID != l.OwnerID {
		if err := requirePerm(op, domain.PermLeadAssign); err != nil {
			return err
		}
	}
	shootDate := l.ShootDate
	if req.ShootDate != "" {
		shootDate = strPtr(req.ShootDate)
	}
	return s.LeadRepo.Update(ctx, op.CompanyID, id, map[string]interface{}{
		"name":         orDefault(req.Name, l.Name),
		"mobile":       orDefault(req.Mobile, l.Mobile),
		"source":       orDefault(req.Source, l.Source),
		"project_type": orDefault(req.ProjectType, l.ProjectType),
		"budget_min":   orDefaultFloat(req.BudgetMin, l.BudgetMin),
		"budget_max":   orDefaultFloat(req.BudgetMax, l.BudgetMax),
		"status":       orDefaultEnum(req.Status, l.Status),
		"shoot_date":   shootDate,
		"remark":       orDefault(req.Remark, l.Remark),
		"owner_id":     orDefaultInt64(req.OwnerID, l.OwnerID),
		"updated_by":   op.UserID,
	})
}

func (s *Service) FollowLead(ctx context.Context, op Operator, id int64, req dto.LeadFollowReq) error {
	l, err := s.LeadRepo.GetByID(ctx, op.CompanyID, id)
	if err != nil {
		return errs.NotFound(errs.ErrLeadNotFound)
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	return s.LeadRepo.Update(ctx, op.CompanyID, id, map[string]interface{}{
		"follower":       l.Follower + 1,
		"last_follow_at": now,
		"next_follow_at": nil,
		"updated_by":     op.UserID,
	})
}

// ConvertLeadToCustomer 线索转客户。
//
// 幂等：线索已关联客户（新增线索时自动建档，或此前已转过一次）直接复用并返回，
// 不再重复建档 —— crm_customer.mobile 只有普通索引、无唯一约束，
// 重复转只会在客户表留下同名同号的两条记录，后续按手机号查档就只剩一条能被命中。
func (s *Service) ConvertLeadToCustomer(ctx context.Context, op Operator, leadID int64) (*model.Customer, error) {
	l, err := s.LeadRepo.GetByID(ctx, op.CompanyID, leadID)
	if err != nil {
		return nil, errs.NotFound(errs.ErrLeadNotFound)
	}

	var c *model.Customer
	err = repository.Tx(func(tx *gorm.DB) error {
		if l.CustomerID > 0 {
			existing, gerr := s.CustomerRepo.WithTx(tx).GetByID(ctx, op.CompanyID, l.CustomerID)
			switch {
			case gerr == nil:
				c = existing
				// 客户已存在 ≠ 线索已推进：仍要落「已成交」，
				// 否则重复转客户只会拿回客户，线索状态永远停在待回复。
				return s.LeadRepo.WithTx(tx).Update(ctx, op.CompanyID, leadID, map[string]interface{}{
					"status": enum.LeadStatusConfirmed,
				})
			case errors.Is(gerr, gorm.ErrRecordNotFound):
				// 客户已被删除，继续走下面的建档兜底
			default:
				return errs.Internal("")
			}
		}

		created, cerr := s.findOrCreateCustomerByMobile(ctx, s.CustomerRepo.WithTx(tx),
			op.CompanyID, l.Name, l.Mobile, orDefault(l.Source, CustomerSourceLead))
		if cerr != nil {
			return cerr
		}
		if created == nil {
			// 手机号为空：不建无手机号的客户（宁缺毋滥），引导先补全手机号
			return errs.BadRequest("线索缺少手机号，无法建立客户，请先补全手机号")
		}
		c = created

		return s.LeadRepo.WithTx(tx).Update(ctx, op.CompanyID, leadID, map[string]interface{}{
			"customer_id": c.ID,
			"status":      enum.LeadStatusConfirmed,
		})
	})
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (s *Service) CreateQuote(ctx context.Context, op Operator, leadID int64, req dto.QuoteCreateReq) (*model.Quote, error) {
	l, err := s.LeadRepo.GetByID(ctx, op.CompanyID, leadID)
	if err != nil {
		return nil, errs.NotFound(errs.ErrLeadNotFound)
	}

	pkg, err := s.PackageRepo.GetByID(ctx, op.CompanyID, req.PackageID)
	if err != nil {
		return nil, errs.NotFound(errs.ErrPackageNotFound)
	}

	q := model.Quote{
		TenantBase: model.TenantBase{
			Base:      model.Base{CreatedBy: op.UserID, UpdatedBy: op.UserID},
			CompanyID: op.CompanyID,
		},
		Code:        domain.GenCode("QT"),
		LeadID:      leadID,
		CustomerID:  l.CustomerID,
		PackageID:   req.PackageID,
		Title:       req.Title,
		PackageName: pkg.Name,
		BasePrice:   pkg.BasePrice,
		AddonPrice:  req.AddonPrice,
		TotalPrice:  pkg.BasePrice + req.AddonPrice,
		ShootDate:   strPtr(req.ShootDate),
		Remark:      req.Remark,
		OwnerID:     op.UserID,
		Status:      enum.QuoteStatusSent,
	}
	if err := s.LeadRepo.CreateQuote(ctx, &q); err != nil {
		return nil, err
	}

	s.LeadRepo.Update(ctx, op.CompanyID, leadID, map[string]interface{}{"status": enum.LeadStatusQuoted})
	// 报价已发出即通知客户（客户 ID 为 0 时内部跳过，不写孤儿通知）
	s.NotifyClient(ctx, op, q.CustomerID, enum.NotificationTypeOrder, "收到新的报价单",
		fmt.Sprintf("报价单 %s 已生成（%.2f 元），请在「我的报价」中查看", q.Code, q.TotalPrice), "quote", q.ID)
	return &q, nil
}

// ListQuotes 线索下的报价历史（倒序，最新在前）
func (s *Service) ListQuotes(ctx context.Context, op Operator, leadID int64) ([]model.Quote, error) {
	if _, err := s.LeadRepo.GetByID(ctx, op.CompanyID, leadID); err != nil {
		return nil, errs.NotFound(errs.ErrLeadNotFound)
	}
	return s.LeadRepo.ListQuotesByLead(ctx, op.CompanyID, leadID)
}

// UpdateQuoteStatus 变更报价单状态（接受/拒绝/成交/撤回为草稿）。
// 报价被接受时同步回写线索状态为「已成交」，保持两端一致。
func (s *Service) UpdateQuoteStatus(ctx context.Context, op Operator, quoteID int64, status enum.QuoteStatus) error {
	q, err := s.LeadRepo.GetQuoteByID(ctx, op.CompanyID, quoteID)
	if err != nil {
		return errs.NotFound(errs.ErrQuoteNotFound)
	}
	if status == 0 {
		return errs.BadRequest("报价状态不能为空")
	}
	updates := map[string]interface{}{
		"status":     status,
		"updated_by": op.UserID,
	}
	if status == enum.QuoteStatusAccepted {
		updates["accept_at"] = time.Now().Format("2006-01-02 15:04:05")
	}
	if err := s.LeadRepo.UpdateQuote(ctx, op.CompanyID, quoteID, updates); err != nil {
		return err
	}
	if status == enum.QuoteStatusAccepted && q.LeadID > 0 {
		_ = s.LeadRepo.Update(ctx, op.CompanyID, q.LeadID, map[string]interface{}{
			"status":     enum.LeadStatusConfirmed,
			"updated_by": op.UserID,
		})
	}
	return nil
}
