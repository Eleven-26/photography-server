package repository

import (
	"gorm.io/gorm"
	"photography-server/internal/model"
)

type AssetRepo struct {
	Repo
}

// WithTx 返回绑定到指定事务连接的副本，事务内的所有写操作将复用该连接，
// 保证跨多张表的写入原子性（失败自动回滚）。
func (r *AssetRepo) WithTx(tx *gorm.DB) *AssetRepo {
	return &AssetRepo{Repo: Repo{db: tx}}
}

func NewAssetRepo() *AssetRepo {
	return &AssetRepo{}
}

// List 作品列表（分页 + 关键字 + 类别 + 状态筛选）
func (r *AssetRepo) List(companyID int64, page, pageSize int, keyword, category, status string) ([]model.Asset, int64, error) {
	q := r.tenant(companyID)
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

// GetByID 根据 ID 查询作品
func (r *AssetRepo) GetByID(companyID, id int64) (*model.Asset, error) {
	var a model.Asset
	if err := r.tenant(companyID).First(&a, id).Error; err != nil {
		return nil, err
	}
	return &a, nil
}

// Create 创建作品
func (r *AssetRepo) Create(a *model.Asset) error {
	return r.tenant(a.CompanyID).Create(a).Error
}

// Update 更新作品
func (r *AssetRepo) Update(companyID, id int64, updates map[string]interface{}) error {
	return r.tenant(companyID).Model(&model.Asset{}).Where("id = ?", id).Updates(updates).Error
}

// Delete 删除作品
func (r *AssetRepo) Delete(companyID, id int64) error {
	return r.tenant(companyID).Delete(&model.Asset{}, id).Error
}
