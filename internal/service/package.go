package service

import (
	"time"

	"gorm.io/gorm"

	"photography-server/internal/model"
	"photography-server/internal/pkg/errs"
)

type PackageReq struct {
	StoreID        int64   `json:"store_id"`
	Name           string  `json:"name" binding:"required"`
	Cover          string  `json:"cover"`
	Category       string  `json:"category"`
	BasePrice      float64 `json:"base_price" binding:"required"`
	DepositRate    float64 `json:"deposit_rate"`
	PhotosIncluded int     `json:"photos_included"`
	ShootHours     float64 `json:"shoot_hours"`
	ContentDesc    string  `json:"content_desc"`
	AddonUnitPrice float64 `json:"addon_unit_price"`
	Status         string  `json:"status"`
}

func (s *Service) ListPackages(op Operator, page, pageSize int, keyword, status, category string) ([]model.Package, int64, error) {
	q := s.tenant(op)
	if keyword != "" {
		kw := "%" + keyword + "%"
		q = q.Where("name LIKE ? OR code LIKE ?", kw, kw)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if category != "" {
		q = q.Where("category = ?", category)
	}
	var total int64
	if err := q.Model(&model.Package{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.Package
	page, pageSize = normalizePage(page, pageSize)
	if err := q.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func (s *Service) GetPackage(op Operator, id int64) (*model.Package, error) {
	var p model.Package
	if err := s.tenant(op).First(&p, id).Error; err != nil {
		return nil, errs.NotFound("套餐不存在")
	}
	return &p, nil
}

// createPackageTx 创建套餐（含定金计算），tx 可为 nil（使用默认会话）
func (s *Service) createPackageTx(tx *gorm.DB, op Operator, req PackageReq, baseVersion int) (*model.Package, error) {
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
	var err error
	if tx != nil {
		err = tx.Create(&p).Error
	} else {
		err = s.tenant(op).Create(&p).Error
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (s *Service) CreatePackage(op Operator, req PackageReq) (*model.Package, error) {
	return s.createPackageTx(nil, op, req, 0)
}

// UpdatePackage 编辑套餐；若已被订单引用且价格/内容变化，则生成新版本（旧版本保留）
func (s *Service) UpdatePackage(op Operator, id int64, req PackageReq) error {
	var p model.Package
	if err := s.tenant(op).First(&p, id).Error; err != nil {
		return errs.NotFound("套餐不存在")
	}
	var refCount int64
	s.tenant(op).Model(&model.Order{}).Where("package_id = ?", id).Count(&refCount)
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
			_, err := s.createPackageTx(s.tenant(op), op, req, p.Version)
			return err
		}
		// 未变更仅允许更新展示字段
		return s.tenant(op).Model(&p).Updates(map[string]interface{}{
			"cover": req.Cover, "shoot_hours": req.ShootHours,
			"status": req.Status, "updated_by": op.UserID,
		}).Error
	}
	// 未被引用：直接修改
	return s.tenant(op).Model(&p).Updates(map[string]interface{}{
		"store_id": req.StoreID, "name": req.Name, "cover": req.Cover,
		"category": req.Category, "base_price": req.BasePrice,
		"deposit_rate":    req.DepositRate,
		"deposit_amt":     round2(req.BasePrice * req.DepositRate / 100),
		"photos_included": req.PhotosIncluded, "shoot_hours": req.ShootHours,
		"content_desc": req.ContentDesc, "addon_unit_price": req.AddonUnitPrice,
		"status": req.Status, "updated_by": op.UserID,
	}).Error
}

func (s *Service) ChangePackageStatus(op Operator, id int64, status string) error {
	var p model.Package
	if err := s.tenant(op).First(&p, id).Error; err != nil {
		return errs.NotFound("套餐不存在")
	}
	updates := map[string]interface{}{"status": status, "updated_by": op.UserID}
	if status == model.PackageStatusActive {
		now := time.Now().Format("2006-01-02 15:04:05")
		updates["published_at"] = now
	}
	return s.tenant(op).Model(&p).Updates(updates).Error
}

func (s *Service) DeletePackage(op Operator, id int64) error {
	var p model.Package
	if err := s.tenant(op).First(&p, id).Error; err != nil {
		return errs.NotFound("套餐不存在")
	}
	if p.Status == model.PackageStatusActive {
		return errs.BadRequest("已上架套餐不可删除，请先下线")
	}
	return s.tenant(op).Delete(&p).Error
}
