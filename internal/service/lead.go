package service

import (
	"context"
	"time"

	"photography-server/internal/domain"
	"photography-server/internal/enum"
	"photography-server/internal/model"
	"photography-server/internal/pkg/errs"
	"photography-server/internal/presentation/dto"
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
	if err := s.LeadRepo.Create(ctx, &l); err != nil {
		return nil, err
	}
	return &l, nil
}

func (s *Service) UpdateLead(ctx context.Context, op Operator, id int64, req dto.LeadUpdateReq) error {
	_, err := s.LeadRepo.GetByID(ctx, op.CompanyID, id)
	if err != nil {
		return errs.NotFound(errs.ErrLeadNotFound)
	}
	return s.LeadRepo.Update(ctx, op.CompanyID, id, map[string]interface{}{
		"name":         req.Name,
		"mobile":       req.Mobile,
		"source":       req.Source,
		"project_type": req.ProjectType,
		"budget_min":   req.BudgetMin,
		"budget_max":   req.BudgetMax,
		"status":       req.Status,
		"shoot_date":   req.ShootDate,
		"remark":       req.Remark,
		"owner_id":     req.OwnerID,
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

func (s *Service) ConvertLeadToCustomer(ctx context.Context, op Operator, leadID int64) (*model.Customer, error) {
	l, err := s.LeadRepo.GetByID(ctx, op.CompanyID, leadID)
	if err != nil {
		return nil, errs.NotFound(errs.ErrLeadNotFound)
	}

	c := model.Customer{
		TenantBase: model.TenantBase{
			Base:      model.Base{CreatedBy: op.UserID, UpdatedBy: op.UserID},
			CompanyID: op.CompanyID,
		},
		Code:    domain.GenCode("CU"),
		StoreID: l.StoreID,
		Name:    l.Name,
		Mobile:  l.Mobile,
		Source:  l.Source,
		Status:  enum.CustomerStatusPotential,
	}
	if err := s.CustomerRepo.Create(ctx, &c); err != nil {
		return nil, err
	}

	s.LeadRepo.Update(ctx, op.CompanyID, leadID, map[string]interface{}{
		"customer_id": c.ID,
		"status":      enum.LeadStatusConfirmed,
	})

	return &c, nil
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
	return &q, nil
}
