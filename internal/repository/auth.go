package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"photography-server/internal/model"
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

// GetByUsername 按用户名查用户（登录用）。ctx 透传请求上下文：SQL span 挂到当前链路、支持超时取消。
func (r *AuthRepo) GetByUsername(ctx context.Context, username string) (*model.SysUser, error) {
	var u model.SysUser
	if err := r.conn().WithContext(ctx).Where("username = ?", username).First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

// GetByMobile 按手机号查员工（摄影师 App 验证码登录用）。
// 登录发生在认证之前、尚无公司上下文，故不做租户过滤；同号多租户时取最早注册者。
func (r *AuthRepo) GetByMobile(ctx context.Context, mobile string) (*model.SysUser, error) {
	var u model.SysUser
	if err := r.conn().WithContext(ctx).
		Where("mobile = ? AND deleted = 0", mobile).Order("id ASC").First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

// TouchLogin 登录成功后刷新最近登录时间与 IP
func (r *AuthRepo) TouchLogin(ctx context.Context, userID int64, ip string) error {
	return r.UpdateLoginInfo(ctx, userID, ip)
}

func (r *AuthRepo) GetByID(ctx context.Context, companyID, userID int64) (*model.SysUser, error) {
	var u model.SysUser
	if err := r.tenant(companyID).WithContext(ctx).First(&u, userID).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *AuthRepo) UpdateLoginInfo(ctx context.Context, userID int64, ip string) error {
	now := time.Now().Format("2006-01-02 15:04:05")
	return r.conn().WithContext(ctx).Model(&model.SysUser{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"last_login_at": now,
		"last_login_ip": ip,
	}).Error
}

func (r *AuthRepo) UpdatePassword(ctx context.Context, userID int64, hash string) error {
	return r.conn().WithContext(ctx).Model(&model.SysUser{}).Where("id = ?", userID).Update("password", hash).Error
}
