package repository

import (
	"gorm.io/gorm"

	"photography-server/internal/model"
)

type CustomerRepo struct{}

func NewCustomerRepo() *CustomerRepo {
	return &CustomerRepo{}
}

// List 客户列表（分页 + 关键字 + 状态 + 等级筛选）
func (r *CustomerRepo) List(companyID int64, page, pageSize int, keyword, status, level string) ([]model.Customer, int64, error) {
	q := tenant(companyID)
	if keyword != "" {
		kw := "%" + keyword + "%"
		q = q.Where("name LIKE ? OR mobile LIKE ? OR wechat LIKE ? OR code LIKE ?", kw, kw, kw, kw)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if level != "" {
		q = q.Where("level = ?", level)
	}
	var total int64
	if err := q.Model(&model.Customer{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.Customer
	page, pageSize = normalizePage(page, pageSize)
	if err := q.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// GetByID 根据 ID 查询客户
func (r *CustomerRepo) GetByID(companyID, id int64) (*model.Customer, error) {
	var c model.Customer
	if err := tenant(companyID).First(&c, id).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

// Create 创建客户（支持外部事务）
func (r *CustomerRepo) Create(tx *gorm.DB, c *model.Customer) error {
	return tx.Create(c).Error
}

// Update 更新客户
func (r *CustomerRepo) Update(companyID, id int64, updates map[string]interface{}) error {
	return tenant(companyID).Model(&model.Customer{}).Where("id = ?", id).Updates(updates).Error
}

// Delete 删除客户
func (r *CustomerRepo) Delete(companyID, id int64) error {
	return tenant(companyID).Delete(&model.Customer{}, id).Error
}

// Stats 客户统计
func (r *CustomerRepo) Stats(companyID int64) (total, active, goldUp int64, err error) {
	q := tenant(companyID)
	if err = q.Model(&model.Customer{}).Count(&total).Error; err != nil {
		return
	}
	if err = q.Model(&model.Customer{}).Where("status = ?", model.CustomerStatusActive).Count(&active).Error; err != nil {
		return
	}
	if err = q.Model(&model.Customer{}).Where("level IN ?", []string{model.CustomerLevelGold, model.CustomerLevelPlatinum, model.CustomerLevelDiamond}).Count(&goldUp).Error; err != nil {
		return
	}
	return
}
