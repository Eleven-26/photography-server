package repository

import (
	"gorm.io/gorm"

	"photography-server/internal/common"
)

// Repo 是所有 repository 的公共底座：持有可选的连接句柄。
// 默认（db == nil）时使用注入的默认连接（见 Init）；调用 WithTx 绑定事务后，
// 该副本上的所有方法都会复用同一事务连接，保证业务原子性。
type Repo struct {
	db *gorm.DB
}

// root 默认连接，由组合根（cmd/server/main.go）在启动时经 Init 显式注入一次。
//
// #40 分层纪律：改造前这里直接调用 infrastructure.MySQL() 读取包级单例，
// 使 repository 与基础设施实现硬耦合，无法注入替身、单测必须依赖真实全局状态。
// 现在连接从显式注入点进入；repository 包对 infrastructure 的依赖为零。
//
// 注意：连接不逐个仓储传参而是保留一个"默认连接"入口，是有意为之——
// 项目分层约定「service 只依赖 repository、不持有任何基础设施句柄，开启事务统一走
// repository.Tx(...)」（见 service/service.go 顶部说明）。若把 *gorm.DB 提升为 service 字段，
// 反而会破坏该纪律；因此事务入口与默认连接都留在 repository 内，仅由组合根注入。
var root *gorm.DB

// Init 注入默认数据库连接，仅由组合根在启动时调用一次。
// 显式传参而非读取全局单例，使测试可以注入内存库替身（如 sqlite / 事务回滚沙箱）。
func Init(db *gorm.DB) { root = db }

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
	return root.Transaction(fn)
}

// conn 返回当前查询应使用的连接：事务连接优先，否则回落到注入的默认连接
func (r *Repo) conn() *gorm.DB {
	if r != nil && r.db != nil {
		return r.db
	}
	return root
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
