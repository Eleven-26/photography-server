package repository

import (
	"gorm.io/gorm"

	"photography-server/internal/model"
)

type PackageRepo struct{}

func NewPackageRepo() *PackageRepo {
	return &PackageRepo{}
}

// List 套餐列表（分页 + 关键字 + 状态 + 类别筛选）
func (r *PackageRepo) List(companyID int64, page, pageSize int, keyword, status, category string) ([]model.Package, int64, error) {
	q := tenant(companyID)
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

// GetByID 根据 ID 查询套餐
func (r *PackageRepo) GetByID(companyID, id int64) (*model.Package, error) {
	var p model.Package
	if err := tenant(companyID).First(&p, id).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

// Create 创建套餐（支持外部事务）
func (r *PackageRepo) Create(tx *gorm.DB, p *model.Package) error {
	if tx != nil {
		return tx.Create(p).Error
	}
	return tenant(p.CompanyID).Create(p).Error
}

// Update 更新套餐
func (r *PackageRepo) Update(companyID, id int64, updates map[string]interface{}) error {
	return tenant(companyID).Model(&model.Package{}).Where("id = ?", id).Updates(updates).Error
}

// Delete 删除套餐
func (r *PackageRepo) Delete(companyID, id int64) error {
	return tenant(companyID).Delete(&model.Package{}, id).Error
}

// CountOrdersByPackage 统计引用该套餐的订单数
func (r *PackageRepo) CountOrdersByPackage(companyID, packageID int64) (int64, error) {
	var count int64
	err := tenant(companyID).Model(&model.Order{}).Where("package_id = ?", packageID).Count(&count).Error
	return count, err
}
