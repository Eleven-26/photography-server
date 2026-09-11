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

func (s *Service) ListAssets(ctx context.Context, op Operator, page, pageSize int, keyword, category, status, featured string) ([]model.Asset, int64, error) {
	return s.AssetRepo.List(ctx, op.CompanyID, page, pageSize, keyword, category, status, featured)
}

func (s *Service) GetAsset(ctx context.Context, op Operator, id int64) (*model.Asset, error) {
	a, err := s.AssetRepo.GetByID(ctx, op.CompanyID, id)
	if err != nil {
		return nil, errs.NotFound(errs.ErrAssetNotFound)
	}
	return a, nil
}

func (s *Service) CreateAsset(ctx context.Context, op Operator, req dto.AssetCreateReq) (*model.Asset, error) {
	a := model.Asset{
		TenantBase: model.TenantBase{
			Base:      model.Base{CreatedBy: op.UserID, UpdatedBy: op.UserID},
			CompanyID: op.CompanyID,
		},
		Code:          domain.GenCode("WK"),
		StoreID:       op.StoreID,
		Title:         req.Title,
		Category:      req.Category,
		Cover:         req.Cover,
		Images:        req.Images,
		Description:   req.Description,
		Photographer:  req.Photographer,
		Model:         req.Model,
		Location:      req.Location,
		ShootDate:     strPtr(req.ShootDate),
		PackageIDs:    req.PackageIDs,
		Status:        orDefaultEnum(req.Status, enum.AssetStatusDraft),
		Visibility:    orDefaultInt(req.Visibility, enum.AssetVisibilityPublic),
		Featured:      req.Featured,
		Authorization: orDefaultInt(req.Authorization, enum.AssetAuthGranted),
	}
	if a.Status == enum.AssetStatusPublished {
		a.PublishedAt = strPtr(time.Now().Format("2006-01-02 15:04:05"))
	}
	if err := s.AssetRepo.Create(ctx, &a); err != nil {
		return nil, err
	}
	return &a, nil
}

func (s *Service) UpdateAsset(ctx context.Context, op Operator, id int64, req dto.AssetUpdateReq) error {
	a, err := s.AssetRepo.GetByID(ctx, op.CompanyID, id)
	if err != nil {
		return errs.NotFound(errs.ErrAssetNotFound)
	}
	updates := map[string]interface{}{
		"title":        req.Title,
		"category":     req.Category,
		"cover":        req.Cover,
		"images":       req.Images,
		"description":  req.Description,
		"photographer": req.Photographer,
		"model":        req.Model,
		"location":     req.Location,
		"shoot_date":   strPtr(req.ShootDate),
		"package_ids":  req.PackageIDs,
		"updated_by":   op.UserID,
	}
	// 指针字段：nil = 未传（保持原值），非 nil = 显式赋值（0 也生效）
	if req.Status != nil {
		updates["status"] = *req.Status
		if *req.Status == enum.AssetStatusPublished && a.Status != enum.AssetStatusPublished {
			updates["published_at"] = time.Now().Format("2006-01-02 15:04:05")
		}
	}
	if req.Visibility != nil {
		updates["visibility"] = *req.Visibility
	}
	if req.Featured != nil {
		updates["featured"] = *req.Featured
	}
	if req.Authorization != nil {
		updates["authorization"] = *req.Authorization
	}
	return s.AssetRepo.Update(ctx, op.CompanyID, id, updates)
}

// UpdateAssetFlags 轻量开关：发布状态 / 公开可见性 / 精选展示。
// 与 UpdateAsset 分离，避免"只想取消精选"却必须回传全部字段（title 等为 required）。
func (s *Service) UpdateAssetFlags(ctx context.Context, op Operator, id int64, req dto.AssetFlagsReq) error {
	a, err := s.AssetRepo.GetByID(ctx, op.CompanyID, id)
	if err != nil {
		return errs.NotFound(errs.ErrAssetNotFound)
	}
	if req.Status == nil && req.Visibility == nil && req.Featured == nil {
		return errs.BadRequest("未提供任何需要变更的字段")
	}
	updates := map[string]interface{}{"updated_by": op.UserID}
	if req.Status != nil {
		if *req.Status != enum.AssetStatusDraft && *req.Status != enum.AssetStatusPublished {
			return errs.BadRequest("作品状态取值非法")
		}
		updates["status"] = *req.Status
		if *req.Status == enum.AssetStatusPublished && a.PublishedAt == nil {
			updates["published_at"] = time.Now().Format("2006-01-02 15:04:05")
		}
	}
	if req.Visibility != nil {
		if *req.Visibility != enum.AssetVisibilityPublic && *req.Visibility != enum.AssetVisibilityPrivate {
			return errs.BadRequest("可见性取值非法")
		}
		updates["visibility"] = *req.Visibility
	}
	if req.Featured != nil {
		if *req.Featured != 0 && *req.Featured != 1 {
			return errs.BadRequest("精选取值非法")
		}
		updates["featured"] = *req.Featured
	}
	return s.AssetRepo.Update(ctx, op.CompanyID, id, updates)
}

func (s *Service) PublishAsset(ctx context.Context, op Operator, id int64) error {
	return s.AssetRepo.Update(ctx, op.CompanyID, id, map[string]interface{}{
		"status":     enum.AssetStatusPublished,
		"updated_by": op.UserID,
	})
}

func (s *Service) DeleteAsset(ctx context.Context, op Operator, id int64) error {
	return s.AssetRepo.Delete(ctx, op.CompanyID, id)
}
