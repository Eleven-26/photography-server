package repository

import (
	"gorm.io/gorm"

	"photography-server/internal/common"
	"photography-server/internal/infrastructure"
)

// tenant 返回按 company_id 过滤的查询会话，实现 SaaS 多租户隔离
func tenant(companyID int64) *gorm.DB {
	return infrastructure.MySQL().Where("company_id = ?", companyID)
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
