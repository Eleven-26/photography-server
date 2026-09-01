package repository

import (
	"photography-server/internal/infrastructure"
	"photography-server/internal/model"
)

type UserRepo struct{}

func NewUserRepo() *UserRepo { return &UserRepo{} }

func (r *UserRepo) List(companyID int64, page, pageSize int, keyword string, storeID int64) ([]model.SysUser, int64, error) {
	q := tenant(companyID)
	if keyword != "" {
		kw := "%" + keyword + "%"
		q = q.Where("username LIKE ? OR nickname LIKE ? OR mobile LIKE ?", kw, kw, kw)
	}
	if storeID > 0 {
		q = q.Where("store_id = ?", storeID)
	}
	var total int64
	if err := q.Model(&model.SysUser{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.SysUser
	page, pageSize = normalizePage(page, pageSize)
	if err := q.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func (r *UserRepo) GetByID(companyID, userID int64) (*model.SysUser, error) {
	var u model.SysUser
	if err := tenant(companyID).First(&u, userID).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepo) CountByUsername(companyID int64, username string) (int64, error) {
	var count int64
	err := tenant(companyID).Model(&model.SysUser{}).Where("username = ?", username).Count(&count).Error
	return count, err
}

func (r *UserRepo) Create(u *model.SysUser) error {
	return infrastructure.MySQL().Create(u).Error
}

func (r *UserRepo) Update(companyID, userID int64, updates map[string]interface{}) error {
	return tenant(companyID).Model(&model.SysUser{}).Where("id = ?", userID).Updates(updates).Error
}

func (r *UserRepo) Delete(companyID, userID int64) error {
	return tenant(companyID).Delete(&model.SysUser{}, userID).Error
}

func (r *UserRepo) UpdatePassword(companyID, userID int64, hash string) error {
	return tenant(companyID).Model(&model.SysUser{}).Where("id = ?", userID).Update("password", hash).Error
}

// -------- 角色 --------

func (r *UserRepo) ListRoles(companyID int64) ([]model.SysRole, error) {
	var list []model.SysRole
	err := tenant(companyID).Order("id ASC").Find(&list).Error
	return list, err
}

func (r *UserRepo) CreateRole(role *model.SysRole) error {
	return infrastructure.MySQL().Create(role).Error
}

func (r *UserRepo) UpdateRole(companyID, roleID int64, updates map[string]interface{}) error {
	return tenant(companyID).Model(&model.SysRole{}).Where("id = ?", roleID).Updates(updates).Error
}

func (r *UserRepo) DeleteRole(companyID, roleID int64) error {
	return tenant(companyID).Delete(&model.SysRole{}, roleID).Error
}

// -------- 门店 --------

func (r *UserRepo) ListStores(companyID int64) ([]model.SysStore, error) {
	var list []model.SysStore
	err := tenant(companyID).Order("id ASC").Find(&list).Error
	return list, err
}

func (r *UserRepo) CreateStore(store *model.SysStore) error {
	return infrastructure.MySQL().Create(store).Error
}

func (r *UserRepo) UpdateStore(companyID, storeID int64, updates map[string]interface{}) error {
	return tenant(companyID).Model(&model.SysStore{}).Where("id = ?", storeID).Updates(updates).Error
}

func (r *UserRepo) DeleteStore(companyID, storeID int64) error {
	return tenant(companyID).Delete(&model.SysStore{}, storeID).Error
}
