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

func (s *Service) ListAssets(op Operator, page, pageSize int, keyword, category, status string) ([]model.Asset, int64, error) {
	return s.AssetRepo.List(op.CompanyID, page, pageSize, keyword, category, status)
}

func (s *Service) GetAsset(op Operator, id int64) (*model.Asset, error) {
	a, err := s.AssetRepo.GetByID(op.CompanyID, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.NotFound(common.ErrAssetNotFound)
		}
		return nil, err
	}
	return a, nil
}

func (s *Service) CreateAsset(op Operator, req dto.AssetCreateReq) (*model.Asset, error) {
	a := model.Asset{
		TenantBase: model.TenantBase{
			Base:      model.Base{CreatedBy: op.UserID, UpdatedBy: op.UserID},
			CompanyID: op.CompanyID,
		},
		Code:         genCode("WK"),
		Title:        req.Title,
		Category:     req.Category,
		Cover:        req.Cover,
		Images:       req.Images,
		Description:  req.Description,
		Photographer: req.Photographer,
		Model:        req.Model,
		Location:     req.Location,
		Status:       orDefault(req.Status, model.AssetStatusDraft),
	}
	if a.Status == model.AssetStatusPublished {
		now := time.Now().Format("2006-01-02 15:04:05")
		a.PublishedAt = &now
	}
	if err := s.AssetRepo.Create(&a); err != nil {
		return nil, err
	}
	return &a, nil
}

func (s *Service) UpdateAsset(op Operator, id int64, req dto.AssetUpdateReq) error {
	_, err := s.AssetRepo.GetByID(op.CompanyID, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errs.NotFound(common.ErrAssetNotFound)
		}
		return err
	}
	return s.AssetRepo.Update(op.CompanyID, id, map[string]interface{}{
		"title": req.Title, "category": req.Category, "cover": req.Cover,
		"images": req.Images, "description": req.Description,
		"photographer": req.Photographer, "model": req.Model,
		"location": req.Location, "status": req.Status, "updated_by": op.UserID,
	})
}

func (s *Service) DeleteAsset(op Operator, id int64) error {
	return s.AssetRepo.Delete(op.CompanyID, id)
}
