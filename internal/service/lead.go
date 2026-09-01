package service

import (
	"errors"
	"time"

	"gorm.io/gorm"

	"photography-server/internal/common"
	"photography-server/internal/model"
	"photography-server/internal/pkg/errs"
	"photography-server/internal/presentation/dto"
)

func (s *Service) ListLeads(op Operator, page, pageSize int, keyword, status string, ownerID int64) ([]model.Lead, int64, error) {
	return s.LeadRepo.List(op.CompanyID, page, pageSize, keyword, status, ownerID)
}

func (s *Service) GetLead(op Operator, id int64) (*model.Lead, error) {
	l, err := s.LeadRepo.GetByID(op.CompanyID, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.NotFound(common.ErrLeadNotFound)
		}
		return nil, err
	}
	return l, nil
}

func (s *Service) CreateLead(op Operator, req dto.LeadCreateReq) (*model.Lead, error) {
	l := model.Lead{
		TenantBase: model.TenantBase{
			Base:      model.Base{CreatedBy: op.UserID, UpdatedBy: op.UserID},
			CompanyID: op.CompanyID,
		},
		Code:        genCode("LD"),
		StoreID:     req.StoreID,
		Name:        req.Name,
		Mobile:      req.Mobile,
		Source:      req.Source,
		ProjectType: req.ProjectType,
		BudgetMin:   req.BudgetMin,
		BudgetMax:   req.BudgetMax,
		Status:      model.LeadStatusPending,
		ShootDate:   strPtr(req.ShootDate),
		Remark:      req.Remark,
		OwnerID:     orDefaultInt64(req.OwnerID, op.UserID),
	}
	if err := s.LeadRepo.Create(&l); err != nil {
		return nil, err
	}
	return &l, nil
}

func (s *Service) UpdateLead(op Operator, id int64, req dto.LeadUpdateReq) error {
	_, err := s.LeadRepo.GetByID(op.CompanyID, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errs.NotFound(common.ErrLeadNotFound)
		}
		return err
	}
	return s.LeadRepo.Update(op.CompanyID, id, map[string]interface{}{
		"store_id": req.StoreID, "name": req.Name, "mobile": req.Mobile,
		"source": req.Source, "project_type": req.ProjectType,
		"budget_min": req.BudgetMin, "budget_max": req.BudgetMax,
		"status": req.Status, "shoot_date": req.ShootDate,
		"remark": req.Remark, "owner_id": req.OwnerID, "updated_by": op.UserID,
	})
}

func (s *Service) DeleteLead(op Operator, id int64) error {
	return s.LeadRepo.Delete(op.CompanyID, id)
}

// FollowLead 跟进记录：跟进次数+1，更新最近跟进时间
func (s *Service) FollowLead(op Operator, id int64, remark string) error {
	l, err := s.LeadRepo.GetByID(op.CompanyID, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errs.NotFound(common.ErrLeadNotFound)
		}
		return err
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	return s.LeadRepo.Update(op.CompanyID, id, map[string]interface{}{
		"follower":       l.Follower + 1,
		"last_follow_at": now,
		"remark":         remark,
		"updated_by":     op.UserID,
	})
}

// ConvertLeadToCustomer 线索转客户（按手机号查重）
func (s *Service) ConvertLeadToCustomer(op Operator, leadID int64) (*model.Customer, error) {
	l, err := s.LeadRepo.GetByID(op.CompanyID, leadID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.NotFound(common.ErrLeadNotFound)
		}
		return nil, err
	}
	if l.CustomerID > 0 {
		c, err := s.CustomerRepo.GetByID(op.CompanyID, l.CustomerID)
		if err == nil {
			return c, nil
		}
	}
	var customer *model.Customer
	err = s.DB().Transaction(func(tx *gorm.DB) error {
		if l.Mobile != "" {
			exist, err := s.CustomerRepo.GetByMobile(op.CompanyID, l.Mobile)
			if err == nil {
				l.CustomerID = exist.ID
				return s.LeadRepo.Update(op.CompanyID, leadID, map[string]interface{}{"customer_id": exist.ID})
			}
		}
		c, err := s.CreateCustomerTx(tx, op, dto.CustomerCreateReq{
			StoreID: l.StoreID, Name: l.Name, Mobile: l.Mobile,
			Source: l.Source, Remark: l.Remark, Status: model.CustomerStatusPotential,
		})
		if err != nil {
			return err
		}
		customer = c
		l.CustomerID = c.ID
		return s.LeadRepo.Update(op.CompanyID, leadID, map[string]interface{}{"customer_id": c.ID, "updated_by": op.UserID})
	})
	if err != nil {
		return nil, err
	}
	if customer == nil {
		c, err := s.CustomerRepo.GetByID(op.CompanyID, l.CustomerID)
		if err != nil {
			return nil, err
		}
		customer = c
	}
	return customer, nil
}

// -------- 报价单 --------

func (s *Service) CreateQuote(op Operator, leadID int64, req dto.QuoteCreateReq) (*model.Quote, error) {
	l, err := s.LeadRepo.GetByID(op.CompanyID, leadID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.NotFound(common.ErrLeadNotFound)
		}
		return nil, err
	}
	pkg, err := s.PackageRepo.GetByID(op.CompanyID, req.PackageID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.NotFound(common.ErrPackageNotFound)
		}
		return nil, err
	}
	q := model.Quote{
		TenantBase: model.TenantBase{
			Base:      model.Base{CreatedBy: op.UserID, UpdatedBy: op.UserID},
			CompanyID: op.CompanyID,
		},
		Code:        genCode("QT"),
		LeadID:      leadID,
		CustomerID:  l.CustomerID,
		PackageID:   pkg.ID,
		Version:     pkg.Version,
		Title:       orDefault(req.Title, l.Name+"报价"),
		PackageName: pkg.Name,
		BasePrice:   pkg.BasePrice,
		AddonPrice:  req.AddonPrice,
		TotalPrice:  round2(pkg.BasePrice + req.AddonPrice),
		Status:      model.QuoteStatusSent,
		Remark:      req.Remark,
		OwnerID:     l.OwnerID,
		ShootDate:   strPtr(req.ShootDate),
	}
	if err := s.LeadRepo.CreateQuote(&q); err != nil {
		return nil, err
	}
	s.LeadRepo.Update(op.CompanyID, leadID, map[string]interface{}{"status": model.LeadStatusQuoted})
	return &q, nil
}

func (s *Service) ListQuotes(op Operator, leadID int64) ([]model.Quote, error) {
	return s.LeadRepo.ListQuotesByLead(op.CompanyID, leadID)
}

func (s *Service) UpdateQuoteStatus(op Operator, quoteID int64, status string) error {
	_, err := s.LeadRepo.GetQuoteByID(op.CompanyID, quoteID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errs.NotFound(common.ErrQuoteNotFound)
		}
		return err
	}
	return s.LeadRepo.UpdateQuote(op.CompanyID, quoteID, map[string]interface{}{"status": status, "updated_by": op.UserID})
}
