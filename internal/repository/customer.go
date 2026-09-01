package repository

import (
	"photography-server/internal/enum"
	"photography-server/internal/infrastructure"
	"photography-server/internal/model"
	"photography-server/internal/presentation/dto"
)

type CustomerRepo struct{}

func NewCustomerRepo() *CustomerRepo { return &CustomerRepo{} }

func (r *CustomerRepo) List(companyID int64, page, pageSize int, keyword string) ([]model.Customer, int64, error) {
	q := tenant(companyID)
	if keyword != "" {
		kw := "%" + keyword + "%"
		q = q.Where("name LIKE ? OR mobile LIKE ? OR code LIKE ?", kw, kw, kw)
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

func (r *CustomerRepo) GetByID(companyID, customerID int64) (*model.Customer, error) {
	var c model.Customer
	if err := tenant(companyID).First(&c, customerID).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *CustomerRepo) Create(c *model.Customer) error {
	return infrastructure.MySQL().Create(c).Error
}

func (r *CustomerRepo) Update(companyID, customerID int64, updates map[string]interface{}) error {
	return tenant(companyID).Model(&model.Customer{}).Where("id = ?", customerID).Updates(updates).Error
}

func (r *CustomerRepo) Delete(companyID, customerID int64) error {
	return tenant(companyID).Delete(&model.Customer{}, customerID).Error
}

func (r *CustomerRepo) GetStats(companyID int64) (*dto.CustomerStatsResp, error) {
	var resp dto.CustomerStatsResp
	q := tenant(companyID)

	if err := q.Model(&model.Customer{}).Count(&resp.Total).Error; err != nil {
		return nil, err
	}

	if err := q.Model(&model.Customer{}).Where("status = ?", enum.CustomerStatusActive).Count(&resp.Active).Error; err != nil {
		return nil, err
	}

	if err := q.Model(&model.Customer{}).Where("level IN ?", []enum.CustomerLevel{enum.CustomerLevelGold, enum.CustomerLevelPlatinum, enum.CustomerLevelDiamond}).Count(&resp.GoldUp).Error; err != nil {
		return nil, err
	}

	return &resp, nil
}
