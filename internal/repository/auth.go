package repository

import (
	"gorm.io/gorm"
	"photography-server/internal/model"
	"time"
)

type AuthRepo struct {
	Repo
}

// WithTx 返回绑定到指定事务连接的副本，事务内的所有写操作将复用该连接，
// 保证跨多张表的写入原子性（失败自动回滚）。
func (r *AuthRepo) WithTx(tx *gorm.DB) *AuthRepo {
	return &AuthRepo{Repo: Repo{db: tx}}
}

func NewAuthRepo() *AuthRepo { return &AuthRepo{} }

func (r *AuthRepo) GetByUsername(username string) (*model.SysUser, error) {
	var u model.SysUser
	if err := r.conn().Where("username = ?", username).First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *AuthRepo) GetByID(companyID, userID int64) (*model.SysUser, error) {
	var u model.SysUser
	if err := r.tenant(companyID).First(&u, userID).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *AuthRepo) UpdateLoginInfo(userID int64, ip string) error {
	now := time.Now().Format("2006-01-02 15:04:05")
	return r.conn().Model(&model.SysUser{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"last_login_at": now,
		"last_login_ip": ip,
	}).Error
}

func (r *AuthRepo) UpdatePassword(userID int64, hash string) error {
	return r.conn().Model(&model.SysUser{}).Where("id = ?", userID).Update("password", hash).Error
}
