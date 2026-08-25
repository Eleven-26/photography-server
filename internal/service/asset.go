package service

import (
	"time"

	"photography-server/internal/model"
	"photography-server/internal/pkg/errs"
)

type AssetReq struct {
	Title        string `json:"title" binding:"required"`
	Category     string `json:"category"`
	Cover        string `json:"cover"`
	Images       string `json:"images"`
	Description  string `json:"description"`
	Photographer string `json:"photographer"`
	Model        string `json:"model"`
	Location     string `json:"location"`
	Status       string `json:"status"`
}

func (s *Service) ListAssets(op Operator, page, pageSize int, keyword, category, status string) ([]model.Asset, int64, error) {
	q := s.tenant(op)
	if keyword != "" {
		kw := "%" + keyword + "%"
		q = q.Where("title LIKE ? OR code LIKE ?", kw, kw)
	}
	if category != "" {
		q = q.Where("category = ?", category)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	var total int64
	if err := q.Model(&model.Asset{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.Asset
	page, pageSize = normalizePage(page, pageSize)
	if err := q.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func (s *Service) GetAsset(op Operator, id int64) (*model.Asset, error) {
	var a model.Asset
	if err := s.tenant(op).First(&a, id).Error; err != nil {
		return nil, errs.NotFound("作品不存在")
	}
	return &a, nil
}

func (s *Service) CreateAsset(op Operator, req AssetReq) (*model.Asset, error) {
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
	if err := s.tenant(op).Create(&a).Error; err != nil {
		return nil, err
	}
	return &a, nil
}

func (s *Service) UpdateAsset(op Operator, id int64, req AssetReq) error {
	var a model.Asset
	if err := s.tenant(op).First(&a, id).Error; err != nil {
		return errs.NotFound("作品不存在")
	}
	return s.tenant(op).Model(&a).Updates(map[string]interface{}{
		"title": req.Title, "category": req.Category, "cover": req.Cover,
		"images": req.Images, "description": req.Description,
		"photographer": req.Photographer, "model": req.Model,
		"location": req.Location, "status": req.Status, "updated_by": op.UserID,
	}).Error
}

func (s *Service) DeleteAsset(op Operator, id int64) error {
	return s.tenant(op).Delete(&model.Asset{}, id).Error
}
