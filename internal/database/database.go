package database

import (
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"photography-server/internal/config"
)

func New(c *config.Config) (*gorm.DB, error) {
	cfg := &gorm.Config{}
	if c.DB.LogMode {
		cfg.Logger = logger.Default.LogMode(logger.Info)
	}

	db, err := gorm.Open(mysql.Open(c.DB.DSN()), cfg)
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxIdleConns(c.DB.MaxIdleConns)
	sqlDB.SetMaxOpenConns(c.DB.MaxOpenConns)
	return db, nil
}

func Ping(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Ping()
}
