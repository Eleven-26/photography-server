package service

import (
	"photography-server/internal/common"
	"photography-server/internal/enum"
	"photography-server/internal/model"
	"photography-server/internal/pkg/errs"
	"photography-server/internal/presentation/dto"
)

func (s *Service) ListPackages(op Operator, page, pageSize int, keyword, status, category string) ([]model.Package, int64, error) {
	return s.PackageRepo.List(op.CompanyID, page, pageSize, keyword, status, category)
}

func (s *Service) GetPackage(op Operator, id int64) (*model.Package, error) {
	p, err := s.PackageRepo.GetByID(op.CompanyID, id)
	if err != nil {
		return nil, errs.NotFound(common.ErrPackageNotFound)
	}
	return p, nil
}

func (s *Service) CreatePackage(op Operator, req dto.PackageReq) (*model.Package, error) {
	p := model.Package{
		TenantBase: model.TenantBase{
			Base:      model.Base{CreatedBy: op.UserID, UpdatedBy: op.UserID},
			CompanyID: op.CompanyID,
		},
		Code:           genCode("PK"),
		StoreID:        orDefaultInt64(req.StoreID, op.StoreID),
		Name:           req.Name,
		Cover:          req.Cover,
		Category:       req.Category,
		BasePrice:      req.BasePrice,
		DepositRate:    req.DepositRate,
		DepositAmt:     round2(req.BasePrice * req.DepositRate / 100),
		PhotosIncluded: req.PhotosIncluded,
		ShootHours:     req.ShootHours,
		ContentDesc:    req.ContentDesc,
		AddonUnitPrice: req.AddonUnitPrice,
		Status:         enum.PackageStatusDraft,
		Version:        1,
	}
	if err := s.PackageRepo.Create(&p); err != nil {
		return nil, err
	}
	return &p, nil
}

func (s *Service) UpdatePackage(op Operator, id int64, req dto.PackageReq) error {
	p, err := s.PackageRepo.GetByID(op.CompanyID, id)
	if err != nil {
		return errs.NotFound(common.ErrPackageNotFound)
	}
	if p.Status == enum.PackageStatusActive {
		return errs.BadRequest(common.ErrPackageActiveDelete)
	}
	return s.PackageRepo.Update(op.CompanyID, id, map[string]interface{}{
		"name":             req.Name,
		"cover":            req.Cover,
		"category":         req.Category,
		"base_price":       req.BasePrice,
		"deposit_rate":     req.DepositRate,
		"deposit_amt":      round2(req.BasePrice * req.DepositRate / 100),
		"photos_included":  req.PhotosIncluded,
		"shoot_hours":      req.ShootHours,
		"content_desc":     req.ContentDesc,
		"addon_unit_price": req.AddonUnitPrice,
		"status":           req.Status,
		"updated_by":       op.UserID,
	})
}

func (s *Service) PublishPackage(op Operator, id int64) error {
	p, err := s.PackageRepo.GetByID(op.CompanyID, id)
	if err != nil {
		return errs.NotFound(common.ErrPackageNotFound)
	}
	if p.Status == enum.PackageStatusActive {
		return errs.BadRequest(common.ErrPackageActiveDelete)
	}

	count, _ := s.PackageRepo.CountOrdersByPackage(op.CompanyID, id)
	if count > 0 {
		newP := *p
		newP.ID = 0
		newP.Code = genCode("PK")
		newP.Version = p.Version + 1
		newP.BaseVersion = p.Version
		newP.Status = enum.PackageStatusActive
		newP.CreatedBy = op.UserID
		newP.UpdatedBy = op.UserID
		if err := s.PackageRepo.Create(&newP); err != nil {
			return err
		}
		return s.PackageRepo.Update(op.CompanyID, id, map[string]interface{}{
			"status":     enum.PackageStatusOffline,
			"updated_by": op.UserID,
		})
	}

	return s.PackageRepo.Update(op.CompanyID, id, map[string]interface{}{
		"status":       enum.PackageStatusActive,
		"published_at": "2006-01-02 15:04:05",
		"updated_by":   op.UserID,
	})
}

func (s *Service) OfflinePackage(op Operator, id int64) error {
	p, err := s.PackageRepo.GetByID(op.CompanyID, id)
	if err != nil {
		return errs.NotFound(common.ErrPackageNotFound)
	}
	if p.Status == enum.PackageStatusActive {
		return errs.BadRequest(common.ErrPackageActiveDelete)
	}
	return s.PackageRepo.Update(op.CompanyID, id, map[string]interface{}{
		"status":     enum.PackageStatusOffline,
		"updated_by": op.UserID,
	})
}

func (s *Service) DeletePackage(op Operator, id int64) error {
	p, err := s.PackageRepo.GetByID(op.CompanyID, id)
	if err != nil {
		return errs.NotFound(common.ErrPackageNotFound)
	}
	if p.Status == enum.PackageStatusActive {
		return errs.BadRequest(common.ErrPackageActiveDelete)
	}
	return s.PackageRepo.Delete(op.CompanyID, id)
}
