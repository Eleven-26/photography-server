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

func (s *Service) ListPackages(op Operator, page, pageSize int, keyword, status, category string) ([]model.Package, int64, error) {
	return s.PackageRepo.List(op.CompanyID, page, pageSize, keyword, status, category)
}

func (s *Service) GetPackage(op Operator, id int64) (*model.Package, error) {
	p, err := s.PackageRepo.GetByID(op.CompanyID, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.NotFound(common.ErrPackageNotFound)
		}
		return nil, err
	}
	return p, nil
}

// createPackageTx 创建套餐（含定金计算），tx 可为 nil（使用默认会话）
func (s *Service) createPackageTx(tx *gorm.DB, op Operator, req dto.PackageReq, baseVersion int) (*model.Package, error) {
	rate := req.DepositRate
	if rate <= 0 {
		rate = 30
	}
	p := model.Package{
		TenantBase: model.TenantBase{
			Base:      model.Base{CreatedBy: op.UserID, UpdatedBy: op.UserID},
			CompanyID: op.CompanyID,
		},
		Code:           genCode("PK"),
		StoreID:        req.StoreID,
		Name:           req.Name,
		Cover:          req.Cover,
		Category:       req.Category,
		BasePrice:      req.BasePrice,
		DepositRate:    rate,
		DepositAmt:     round2(req.BasePrice * rate / 100),
		PhotosIncluded: req.PhotosIncluded,
		ShootHours:     req.ShootHours,
		ContentDesc:    req.ContentDesc,
		AddonUnitPrice: req.AddonUnitPrice,
		Status:         orDefault(req.Status, model.PackageStatusDraft),
		Version:        1,
		BaseVersion:    baseVersion,
	}
	if baseVersion > 0 {
		p.Version = baseVersion + 1
	}
	if p.Status == model.PackageStatusActive {
		now := time.Now().Format("2006-01-02 15:04:05")
		p.PublishedAt = &now
	}
	if err := s.PackageRepo.Create(tx, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

func (s *Service) CreatePackage(op Operator, req dto.PackageReq) (*model.Package, error) {
	return s.createPackageTx(nil, op, req, 0)
}

// UpdatePackage 编辑套餐；若已被订单引用且价格/内容变化，则生成新版本（旧版本保留）
func (s *Service) UpdatePackage(op Operator, id int64, req dto.PackageReq) error {
	p, err := s.PackageRepo.GetByID(op.CompanyID, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errs.NotFound(common.ErrPackageNotFound)
		}
		return err
	}
	refCount, err := s.PackageRepo.CountOrdersByPackage(op.CompanyID, id)
	if err != nil {
		return err
	}
	if refCount > 0 {
		// 已被订单引用：校验是否产生实质变更，若变更则生成新版本
		changed := req.Name != p.Name ||
			req.BasePrice != p.BasePrice ||
			req.DepositRate != p.DepositRate ||
			req.Category != p.Category ||
			req.ContentDesc != p.ContentDesc ||
			req.PhotosIncluded != p.PhotosIncluded ||
			req.AddonUnitPrice != p.AddonUnitPrice
		if changed {
			_, err := s.createPackageTx(s.DB(), op, req, p.Version)
			return err
		}
		// 未变更仅允许更新展示字段
		return s.PackageRepo.Update(op.CompanyID, id, map[string]interface{}{
			"cover": req.Cover, "shoot_hours": req.ShootHours,
			"status": req.Status, "updated_by": op.UserID,
		})
	}
	// 未被引用：直接修改
	return s.PackageRepo.Update(op.CompanyID, id, map[string]interface{}{
		"store_id": req.StoreID, "name": req.Name, "cover": req.Cover,
		"category": req.Category, "base_price": req.BasePrice,
		"deposit_rate":    req.DepositRate,
		"deposit_amt":     round2(req.BasePrice * req.DepositRate / 100),
		"photos_included": req.PhotosIncluded, "shoot_hours": req.ShootHours,
		"content_desc": req.ContentDesc, "addon_unit_price": req.AddonUnitPrice,
		"status": req.Status, "updated_by": op.UserID,
	})
}

func (s *Service) ChangePackageStatus(op Operator, id int64, status string) error {
	_, err := s.PackageRepo.GetByID(op.CompanyID, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errs.NotFound(common.ErrPackageNotFound)
		}
		return err
	}
	updates := map[string]interface{}{"status": status, "updated_by": op.UserID}
	if status == model.PackageStatusActive {
		now := time.Now().Format("2006-01-02 15:04:05")
		updates["published_at"] = now
	}
	return s.PackageRepo.Update(op.CompanyID, id, updates)
}

func (s *Service) DeletePackage(op Operator, id int64) error {
	p, err := s.PackageRepo.GetByID(op.CompanyID, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errs.NotFound(common.ErrPackageNotFound)
		}
		return err
	}
	if p.Status == model.PackageStatusActive {
		return errs.BadRequest(common.ErrPackageActiveDelete)
	}
	return s.PackageRepo.Delete(op.CompanyID, id)
}
