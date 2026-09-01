package repository

import (
	"time"

	"photography-server/internal/infrastructure"
	"photography-server/internal/model"
)

type AuthRepo struct{}

func NewAuthRepo() *AuthRepo { return &AuthRepo{} }

func (r *AuthRepo) GetByUsername(username string) (*model.SysUser, error) {
	var u model.SysUser
	if err := infrastructure.MySQL().Where("username = ?", username).First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *AuthRepo) GetByID(companyID, userID int64) (*model.SysUser, error) {
	var u model.SysUser
	if err := tenant(companyID).First(&u, userID).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *AuthRepo) UpdateLoginInfo(userID int64, ip string) error {
	now := time.Now().Format("2006-01-02 15:04:05")
	return infrastructure.MySQL().Model(&model.SysUser{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"last_login_at": now,
		"last_login_ip": ip,
	}).Error
}

func (r *AuthRepo) UpdatePassword(userID int64, hash string) error {
	return infrastructure.MySQL().Model(&model.SysUser{}).Where("id = ?", userID).Update("password", hash).Error
}
