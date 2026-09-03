package repository

import (
	"gorm.io/gorm"

	"photography-server/internal/common"
	"photography-server/internal/infrastructure"
)

// Repo 是所有 repository 的公共底座：持有可选的连接句柄。
// 默认（db == nil）时使用全局 MySQL 单例；调用 WithTx 绑定事务后，
// 该副本上的所有方法都会复用同一事务连接，保证业务原子性。
type Repo struct {
	db *gorm.DB
}

// WithTx 返回绑定到指定事务连接的副本，事务内的所有写操作将共用该连接。
// 用法：在 repository.Tx(func(tx *gorm.DB) error {...}) 回调内，
// 用 repo.WithTx(tx).Xxx(...) 替代 repo.Xxx(...)。
func (r *Repo) WithTx(tx *gorm.DB) *Repo {
	return &Repo{db: tx}
}

// Tx 开启一个数据库事务，是业务层开启事务的唯一入口（service 不再持有 DB 句柄）。
// 回调内所有写操作必须使用 repo.WithTx(tx).Xxx(...) 透传同一连接，
// 保证跨多张表的写入原子性（任一步返回 error 则整体回滚）。
func Tx(fn func(tx *gorm.DB) error) error {
	return infrastructure.MySQL().Transaction(fn)
}

// conn 返回当前查询应使用的连接：事务连接优先，否则回落到全局单例
func (r *Repo) conn() *gorm.DB {
	if r != nil && r.db != nil {
		return r.db
	}
	return infrastructure.MySQL()
}

// tenant 返回按 company_id 过滤的查询会话，实现 SaaS 多租户隔离
func (r *Repo) tenant(companyID int64) *gorm.DB {
	return r.conn().Where("company_id = ?", companyID)
}

// normalizePage 校正分页参数
func normalizePage(page, pageSize int) (int, int) {
	if page <= 0 {
		page = common.DefaultPage
	}
	if pageSize <= 0 {
		pageSize = common.DefaultPageSize
	}
	if pageSize > common.MaxPageSize {
		pageSize = common.MaxPageSize
	}
	return page, pageSize
}
