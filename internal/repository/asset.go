package repository

import (
	"context"

	"gorm.io/gorm"

	"photography-server/internal/enum"
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

// List 作品列表（分页 + 关键字 + 类别 + 状态 + 精选筛选）
func (r *AssetRepo) List(ctx context.Context, companyID int64, page, pageSize int, keyword, category, status, featured string) ([]model.Asset, int64, error) {
	// 行级数据权限：本门店 / 仅本人（created_by）
	q := applyScope(r.tenant(companyID).WithContext(ctx), opOf(ctx), scopeAsset)
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
	if featured != "" {
		q = q.Where("featured = ?", featured)
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
func (r *AssetRepo) GetByID(ctx context.Context, companyID, id int64) (*model.Asset, error) {
	var a model.Asset
	if err := r.tenant(companyID).WithContext(ctx).First(&a, id).Error; err != nil {
		return nil, err
	}
	return &a, nil
}

// ListPublic 客户侧公开作品列表（H5/小程序预约主页的「作品集」）。
// 与 PC 端 List 分开实现：公开接口的可见范围必须由服务端写死（已发布 + 公开），
// 不能复用带 status/featured 外部入参的 PC 查询——否则后续给 PC 加筛选条件会顺带放大公开数据范围。
func (r *AssetRepo) ListPublic(ctx context.Context, companyID int64, page, pageSize int, category string, featuredOnly bool) ([]model.Asset, int64, error) {
	q := r.tenant(companyID).WithContext(ctx).
		Where("status = ? AND visibility = ?", int(enum.AssetStatusPublished), enum.AssetVisibilityPublic)
	if category != "" {
		q = q.Where("category = ?", category)
	}
	if featuredOnly {
		q = q.Where("featured = ?", 1)
	}
	var total int64
	if err := q.Model(&model.Asset{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.Asset
	page, pageSize = normalizePage(page, pageSize)
	// 精选置顶（主页顶部优先），同组内新的在前
	if err := q.Order("featured DESC, id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// IncrViewCount 作品浏览数 +1（公开详情页调用）。
// 用列自增表达式而非「读值 + 写回」，避免并发下的丢更新。
func (r *AssetRepo) IncrViewCount(ctx context.Context, companyID, id int64) error {
	return r.tenant(companyID).WithContext(ctx).Model(&model.Asset{}).
		Where("id = ?", id).UpdateColumn("view_count", gorm.Expr("view_count + 1")).Error
}

// Create 创建作品
func (r *AssetRepo) Create(ctx context.Context, a *model.Asset) error {
	return r.tenant(a.CompanyID).WithContext(ctx).Create(a).Error
}

// Update 更新作品
func (r *AssetRepo) Update(ctx context.Context, companyID, id int64, updates map[string]interface{}) error {
	return r.tenant(companyID).WithContext(ctx).Model(&model.Asset{}).Where("id = ?", id).Updates(updates).Error
}

// Delete 删除作品
func (r *AssetRepo) Delete(ctx context.Context, companyID, id int64) error {
	return r.tenant(companyID).WithContext(ctx).Delete(&model.Asset{}, id).Error
}
