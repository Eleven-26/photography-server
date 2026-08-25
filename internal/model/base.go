package model

import (
	"time"

	"gorm.io/plugin/soft_delete"
)

// Base 所有业务表统一固定字段：创建人/创建时间/修改人/修改时间/是否删除(软删除)
type Base struct {
	ID        int64                 `gorm:"primaryKey;autoIncrement;comment:主键ID" json:"id"`
	CreatedBy int64                 `gorm:"column:created_by;index;comment:创建人" json:"created_by"`
	CreatedAt time.Time             `gorm:"column:created_at;comment:创建时间" json:"created_at"`
	UpdatedBy int64                 `gorm:"column:updated_by;comment:修改人" json:"updated_by"`
	UpdatedAt time.Time             `gorm:"column:updated_at;comment:修改时间" json:"updated_at"`
	Deleted   soft_delete.DeletedAt `gorm:"column:deleted;index;comment:是否删除 0-否 1-是" json:"deleted"`
}

// TenantBase SaaS 多租户基础结构，所有业务表均含 company_id，未来支持一公司多门店、多门店多管理员
type TenantBase struct {
	Base
	CompanyID int64 `gorm:"column:company_id;index;comment:公司ID" json:"company_id"`
}
