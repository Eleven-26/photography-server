package infrastructure

import (
	"sync"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/plugin/opentelemetry/tracing"

	applog "photography-server/internal/pkg/logger"

	"photography-server/internal/config"
)

var (
	dbInstance *gorm.DB
	dbOnce     sync.Once
	dbErr      error
)

// InitMySQL 初始化 MySQL 单例
func InitMySQL(c *config.Config) error {
	dbOnce.Do(func() {
		cfg := &gorm.Config{}
		if c.DB.LogMode {
			cfg.Logger = logger.Default.LogMode(logger.Info)
		}
		dbInstance, dbErr = gorm.Open(mysql.Open(c.DB.DSN()), cfg)
		if dbErr != nil {
			return
		}
		sqlDB, err := dbInstance.DB()
		if err != nil {
			dbErr = err
			return
		}
		sqlDB.SetMaxIdleConns(c.DB.MaxIdleConns)
		sqlDB.SetMaxOpenConns(c.DB.MaxOpenConns)

		// SQL 埋点：GORM 官方 OTel 插件，每条 SQL 产生一个 client span，
		// 语句经 Dialector.Explain 填充实际参数后记录（db.query.text），错误 SQL 会标记 Error 状态。
		// 前提：全局 TracerProvider 已就绪 —— main.go 中 InitSkyWalking 必须先于 InitMySQL 执行。
		// 已知限制：SQL span 挂在 gorm Statement.Context 下，业务调用未透传请求 ctx 时
		// 会以独立 trace 入库（SQL 与参数仍可查，但不挂在 HTTP 请求链路下）。
		if SkyWalkingEnabled() {
			if err := dbInstance.Use(tracing.NewPlugin(tracing.WithoutMetrics())); err != nil {
				applog.Warnf("gorm tracing plugin install failed, sql spans disabled: %v", err)
			}
		}
	})
	return dbErr
}

// MySQL 获取 MySQL 单例
func MySQL() *gorm.DB {
	return dbInstance
}

// Ping 检测 MySQL 连通性
func Ping() error {
	sqlDB, err := dbInstance.DB()
	if err != nil {
		return err
	}
	return sqlDB.Ping()
}
