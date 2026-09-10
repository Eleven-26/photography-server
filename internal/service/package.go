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

func (s *Service) ListPackages(ctx context.Context, op Operator, page, pageSize int, keyword, status, category string) ([]model.Package, int64, error) {
	return s.PackageRepo.List(ctx, op.CompanyID, page, pageSize, keyword, status, category)
}

func (s *Service) GetPackage(ctx context.Context, op Operator, id int64) (*model.Package, error) {
	p, err := s.PackageRepo.GetByID(ctx, op.CompanyID, id)
	if err != nil {
		return nil, errs.NotFound(errs.ErrPackageNotFound)
	}
	return p, nil
}

func (s *Service) CreatePackage(ctx context.Context, op Operator, req dto.PackageReq) (*model.Package, error) {
	p := model.Package{
		TenantBase: model.TenantBase{
			Base:      model.Base{CreatedBy: op.UserID, UpdatedBy: op.UserID},
			CompanyID: op.CompanyID,
		},
		Code:           domain.GenCode("PK"),
		StoreID:        orDefaultInt64(req.StoreID, op.StoreID),
		Name:           req.Name,
		Cover:          req.Cover,
		Category:       req.Category,
		BasePrice:      req.BasePrice,
		DepositRate:    req.DepositRate,
		DepositAmt:     domain.Round2(req.BasePrice * req.DepositRate / 100),
		PhotosIncluded: req.PhotosIncluded,
		ShootHours:     req.ShootHours,
		ContentDesc:    req.ContentDesc,
		AddonUnitPrice: req.AddonUnitPrice,
		Status:         enum.PackageStatusDraft,
		Version:        1,
	}
	if err := s.PackageRepo.Create(ctx, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

func (s *Service) UpdatePackage(ctx context.Context, op Operator, id int64, req dto.PackageReq) error {
	p, err := s.PackageRepo.GetByID(ctx, op.CompanyID, id)
	if err != nil {
		return errs.NotFound(errs.ErrPackageNotFound)
	}
	if p.Status == enum.PackageStatusActive {
		return errs.BadRequest(errs.ErrPackageActiveDelete)
	}
	return s.PackageRepo.Update(ctx, op.CompanyID, id, map[string]interface{}{
		"name":             req.Name,
		"cover":            req.Cover,
		"category":         req.Category,
		"base_price":       req.BasePrice,
		"deposit_rate":     req.DepositRate,
		"deposit_amt":      domain.Round2(req.BasePrice * req.DepositRate / 100),
		"photos_included":  req.PhotosIncluded,
		"shoot_hours":      req.ShootHours,
		"content_desc":     req.ContentDesc,
		"addon_unit_price": req.AddonUnitPrice,
		"status":           orDefaultEnum(req.Status, p.Status),
		"updated_by":       op.UserID,
	})
}

// ChangePackageStatus 套餐状态切换（PC 套餐卡片上架/下线的唯一入口）。
//
// 语义：幂等——目标状态与当前状态一致时直接成功，重复点击或两端并发不会互相报错。
//
// 修复记录：
//   - 原 `OfflinePackage` 守卫写反（`p.Status == Active` 时拒绝），而「下线已上架套餐」
//     恰是该接口唯一的使用场景，等于永远无法下线；
//   - 原 `PublishPackage` 把 Go 时间布局串 "2006-01-02 15:04:05" 原样写进 `published_at`，
//     落库后永远是那一串字面量而非真实时间。
//
// 原 `PublishPackage` 内的「已被订单引用则升版重建」逻辑一并移除：该分支不可达（两方法
// 全仓零调用），且会让普通的上下架开关凭空产生一条重复套餐记录；历史订单已通过
// `package_name / package_version` 快照自保，不受套餐改价影响。
func (s *Service) ChangePackageStatus(ctx context.Context, op Operator, id int64, status enum.PackageStatus) error {
	p, err := s.PackageRepo.GetByID(ctx, op.CompanyID, id)
	if err != nil {
		return errs.NotFound(errs.ErrPackageNotFound)
	}
	switch status {
	case enum.PackageStatusDraft, enum.PackageStatusActive, enum.PackageStatusOffline:
	default:
		return errs.BadRequest(errs.ErrPackageStatusInvalid)
	}
	if p.Status == status {
		return nil // 幂等：已是目标状态
	}
	updates := map[string]interface{}{
		"status":     status,
		"updated_by": op.UserID,
	}
	// 仅首次上架写入上架时间，反复上下架不覆盖首次上架时间
	if status == enum.PackageStatusActive && (p.PublishedAt == nil || *p.PublishedAt == "") {
		updates["published_at"] = time.Now().Format("2006-01-02 15:04:05")
	}
	return s.PackageRepo.Update(ctx, op.CompanyID, id, updates)
}

func (s *Service) DeletePackage(ctx context.Context, op Operator, id int64) error {
	p, err := s.PackageRepo.GetByID(ctx, op.CompanyID, id)
	if err != nil {
		return errs.NotFound(errs.ErrPackageNotFound)
	}
	if p.Status == enum.PackageStatusActive {
		return errs.BadRequest(errs.ErrPackageActiveDelete)
	}
	return s.PackageRepo.Delete(ctx, op.CompanyID, id)
}
