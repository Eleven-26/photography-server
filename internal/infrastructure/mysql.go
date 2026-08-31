package infrastructure

import (
	"sync"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

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
