package service

import (
	"time"

	"gorm.io/gorm"

	"photography-server/internal/model"
	"photography-server/internal/pkg/errs"
)

type LeadCreateReq struct {
	StoreID     int64   `json:"store_id"`
	Name        string  `json:"name" binding:"required"`
	Mobile      string  `json:"mobile"`
	Source      string  `json:"source"`
	ProjectType string  `json:"project_type"`
	BudgetMin   float64 `json:"budget_min"`
	BudgetMax   float64 `json:"budget_max"`
	ShootDate   string  `json:"shoot_date"`
	Remark      string  `json:"remark"`
	OwnerID     int64   `json:"owner_id"`
}

type LeadUpdateReq struct {
	StoreID     int64   `json:"store_id"`
	Name        string  `json:"name"`
	Mobile      string  `json:"mobile"`
	Source      string  `json:"source"`
	ProjectType string  `json:"project_type"`
	BudgetMin   float64 `json:"budget_min"`
	BudgetMax   float64 `json:"budget_max"`
	Status      string  `json:"status"`
	ShootDate   string  `json:"shoot_date"`
	Remark      string  `json:"remark"`
	OwnerID     int64   `json:"owner_id"`
}

func (s *Service) ListLeads(op Operator, page, pageSize int, keyword, status string, ownerID int64) ([]model.Lead, int64, error) {
	q := s.tenant(op)
	if keyword != "" {
		kw := "%" + keyword + "%"
		q = q.Where("name LIKE ? OR mobile LIKE ? OR code LIKE ?", kw, kw, kw)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if ownerID > 0 {
		q = q.Where("owner_id = ?", ownerID)
	}
	var total int64
	if err := q.Model(&model.Lead{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.Lead
	page, pageSize = normalizePage(page, pageSize)
	if err := q.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func (s *Service) GetLead(op Operator, id int64) (*model.Lead, error) {
	var l model.Lead
	if err := s.tenant(op).First(&l, id).Error; err != nil {
		return nil, errs.NotFound("线索不存在")
	}
	return &l, nil
}

func (s *Service) CreateLead(op Operator, req LeadCreateReq) (*model.Lead, error) {
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
	if err := s.tenant(op).Create(&l).Error; err != nil {
		return nil, err
	}
	return &l, nil
}

func (s *Service) UpdateLead(op Operator, id int64, req LeadUpdateReq) error {
	var l model.Lead
	if err := s.tenant(op).First(&l, id).Error; err != nil {
		return errs.NotFound("线索不存在")
	}
	return s.tenant(op).Model(&l).Updates(map[string]interface{}{
		"store_id": req.StoreID, "name": req.Name, "mobile": req.Mobile,
		"source": req.Source, "project_type": req.ProjectType,
		"budget_min": req.BudgetMin, "budget_max": req.BudgetMax,
		"status": req.Status, "shoot_date": req.ShootDate,
		"remark": req.Remark, "owner_id": req.OwnerID, "updated_by": op.UserID,
	}).Error
}

func (s *Service) DeleteLead(op Operator, id int64) error {
	return s.tenant(op).Delete(&model.Lead{}, id).Error
}

// FollowLead 跟进记录：跟进次数+1，更新最近跟进时间
func (s *Service) FollowLead(op Operator, id int64, remark string) error {
	var l model.Lead
	if err := s.tenant(op).First(&l, id).Error; err != nil {
		return errs.NotFound("线索不存在")
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	return s.tenant(op).Model(&l).Updates(map[string]interface{}{
		"follower":       l.Follower + 1,
		"last_follow_at": now,
		"remark":         remark,
		"updated_by":     op.UserID,
	}).Error
}

// ConvertLeadToCustomer 线索转客户（按手机号查重）
func (s *Service) ConvertLeadToCustomer(op Operator, leadID int64) (*model.Customer, error) {
	var l model.Lead
	if err := s.tenant(op).First(&l, leadID).Error; err != nil {
		return nil, errs.NotFound("线索不存在")
	}
	if l.CustomerID > 0 {
		var c model.Customer
		if err := s.tenant(op).First(&c, l.CustomerID).Error; err == nil {
			return &c, nil
		}
	}
	var customer *model.Customer
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		if l.Mobile != "" {
			var exist model.Customer
			if err := tx.Where("company_id = ? AND mobile = ?", op.CompanyID, l.Mobile).First(&exist).Error; err == nil {
				l.CustomerID = exist.ID
				return tx.Model(&l).Update("customer_id", exist.ID).Error
			}
		}
		c, err := s.createCustomerTx(tx, op, CustomerCreateReq{
			StoreID: l.StoreID, Name: l.Name, Mobile: l.Mobile,
			Source: l.Source, Remark: l.Remark, Status: model.CustomerStatusPotential,
		})
		if err != nil {
			return err
		}
		customer = c
		l.CustomerID = c.ID
		return tx.Model(&l).Updates(map[string]interface{}{"customer_id": c.ID, "updated_by": op.UserID}).Error
	})
	if err != nil {
		return nil, err
	}
	if customer == nil {
		var c model.Customer
		if err := s.tenant(op).First(&c, l.CustomerID).Error; err != nil {
			return nil, err
		}
		customer = &c
	}
	return customer, nil
}

// -------- 报价单 --------

type QuoteCreateReq struct {
	PackageID  int64   `json:"package_id" binding:"required"`
	Title      string  `json:"title"`
	AddonPrice float64 `json:"addon_price"`
	ShootDate  string  `json:"shoot_date"`
	Remark     string  `json:"remark"`
}

func (s *Service) CreateQuote(op Operator, leadID int64, req QuoteCreateReq) (*model.Quote, error) {
	var l model.Lead
	if err := s.tenant(op).First(&l, leadID).Error; err != nil {
		return nil, errs.NotFound("线索不存在")
	}
	var pkg model.Package
	if err := s.tenant(op).First(&pkg, req.PackageID).Error; err != nil {
		return nil, errs.NotFound("套餐不存在")
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
	if err := s.tenant(op).Create(&q).Error; err != nil {
		return nil, err
	}
	s.tenant(op).Model(&l).Update("status", model.LeadStatusQuoted)
	return &q, nil
}

func (s *Service) ListQuotes(op Operator, leadID int64) ([]model.Quote, error) {
	var list []model.Quote
	err := s.tenant(op).Where("lead_id = ?", leadID).Order("id DESC").Find(&list).Error
	return list, err
}

func (s *Service) UpdateQuoteStatus(op Operator, quoteID int64, status string) error {
	var q model.Quote
	if err := s.tenant(op).First(&q, quoteID).Error; err != nil {
		return errs.NotFound("报价单不存在")
	}
	return s.tenant(op).Model(&q).Updates(map[string]interface{}{"status": status, "updated_by": op.UserID}).Error
}
