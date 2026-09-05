package repository

import (
	"context"

	"gorm.io/gorm"
	"photography-server/internal/model"
)

type UserRepo struct {
	Repo
}

// WithTx 返回绑定到指定事务连接的副本，事务内的所有写操作将复用该连接，
// 保证跨多张表的写入原子性（失败自动回滚）。
func (r *UserRepo) WithTx(tx *gorm.DB) *UserRepo {
	return &UserRepo{Repo: Repo{db: tx}}
}

func NewUserRepo() *UserRepo { return &UserRepo{} }

func (r *UserRepo) List(ctx context.Context, companyID int64, page, pageSize int, keyword string, storeID int64) ([]model.SysUser, int64, error) {
	q := r.tenant(companyID).WithContext(ctx)
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

func (r *UserRepo) GetByID(ctx context.Context, companyID, userID int64) (*model.SysUser, error) {
	var u model.SysUser
	if err := r.tenant(companyID).WithContext(ctx).First(&u, userID).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepo) CountByUsername(ctx context.Context, companyID int64, username string) (int64, error) {
	var count int64
	err := r.tenant(companyID).WithContext(ctx).Model(&model.SysUser{}).Where("username = ?", username).Count(&count).Error
	return count, err
}

func (r *UserRepo) Create(ctx context.Context, u *model.SysUser) error {
	return r.conn().WithContext(ctx).Create(u).Error
}

func (r *UserRepo) Update(ctx context.Context, companyID, userID int64, updates map[string]interface{}) error {
	return r.tenant(companyID).WithContext(ctx).Model(&model.SysUser{}).Where("id = ?", userID).Updates(updates).Error
}

func (r *UserRepo) Delete(ctx context.Context, companyID, userID int64) error {
	return r.tenant(companyID).WithContext(ctx).Delete(&model.SysUser{}, userID).Error
}

func (r *UserRepo) UpdatePassword(ctx context.Context, companyID, userID int64, hash string) error {
	return r.tenant(companyID).WithContext(ctx).Model(&model.SysUser{}).Where("id = ?", userID).Update("password", hash).Error
}

// -------- 角色 --------

func (r *UserRepo) ListRoles(ctx context.Context, companyID int64) ([]model.SysRole, error) {
	var list []model.SysRole
	err := r.tenant(companyID).WithContext(ctx).Order("id ASC").Find(&list).Error
	return list, err
}

func (r *UserRepo) CreateRole(ctx context.Context, role *model.SysRole) error {
	return r.conn().WithContext(ctx).Create(role).Error
}

func (r *UserRepo) UpdateRole(ctx context.Context, companyID, roleID int64, updates map[string]interface{}) error {
	return r.tenant(companyID).WithContext(ctx).Model(&model.SysRole{}).Where("id = ?", roleID).Updates(updates).Error
}

func (r *UserRepo) DeleteRole(ctx context.Context, companyID, roleID int64) error {
	return r.tenant(companyID).WithContext(ctx).Delete(&model.SysRole{}, roleID).Error
}

// -------- 门店 --------

func (r *UserRepo) ListStores(ctx context.Context, companyID int64) ([]model.SysStore, error) {
	var list []model.SysStore
	err := r.tenant(companyID).WithContext(ctx).Order("id ASC").Find(&list).Error
	return list, err
}

func (r *UserRepo) CreateStore(ctx context.Context, store *model.SysStore) error {
	return r.conn().WithContext(ctx).Create(store).Error
}

func (r *UserRepo) UpdateStore(ctx context.Context, companyID, storeID int64, updates map[string]interface{}) error {
	return r.tenant(companyID).WithContext(ctx).Model(&model.SysStore{}).Where("id = ?", storeID).Updates(updates).Error
}

func (r *UserRepo) DeleteStore(ctx context.Context, companyID, storeID int64) error {
	return r.tenant(companyID).WithContext(ctx).Delete(&model.SysStore{}, storeID).Error
}
