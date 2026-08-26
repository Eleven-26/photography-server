package main

import (
	"time"

	"golang.org/x/crypto/bcrypt"

	"photography-server/internal/model"
	"photography-server/internal/pkg/logger"

	"gorm.io/gorm"
)

// bootstrap 首次启动时初始化默认租户数据（公司/门店/角色/超级管理员）
// 默认账号 admin，密码 admin123456。若 docs/sql/dml.sql 已导入则自动跳过。
func bootstrap(db *gorm.DB) {
	var companyCount int64
	if err := db.Model(&model.SysCompany{}).Count(&companyCount).Error; err == nil && companyCount > 0 {
		logger.Infof("bootstrap skipped: company already exists")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte("admin123456"), bcrypt.DefaultCost)
	if err != nil {
		logger.Errorf("bootstrap fail: generate password: %v", err)
		return
	}

	tx := db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	now := time.Now()
	company := model.SysCompany{
		Base:   model.Base{CreatedBy: 1, CreatedAt: now, UpdatedBy: 1, UpdatedAt: now},
		Name:   "SLOT摄影工作室",
		Status: 1,
	}
	if err := tx.Create(&company).Error; err != nil {
		tx.Rollback()
		logger.Errorf("bootstrap fail: create company: %v", err)
		return
	}

	store := model.SysStore{
		TenantBase: model.TenantBase{
			Base:      model.Base{CreatedBy: 1, CreatedAt: now, UpdatedBy: 1, UpdatedAt: now},
			CompanyID: company.ID,
		},
		Name: "SLOT主门店", Status: 1,
	}
	if err := tx.Create(&store).Error; err != nil {
		tx.Rollback()
		logger.Errorf("bootstrap fail: create store: %v", err)
		return
	}

	base := model.Base{CreatedBy: 1, CreatedAt: now, UpdatedBy: 1, UpdatedAt: now}
	roles := []model.SysRole{
		{TenantBase: model.TenantBase{Base: base, CompanyID: company.ID}, Name: "超级管理员", Code: "admin", Status: 1},
		{TenantBase: model.TenantBase{Base: base, CompanyID: company.ID}, Name: "店长", Code: "manager", Status: 1},
		{TenantBase: model.TenantBase{Base: base, CompanyID: company.ID}, Name: "摄影师", Code: "photographer", Status: 1},
		{TenantBase: model.TenantBase{Base: base, CompanyID: company.ID}, Name: "销售", Code: "sales", Status: 1},
	}
	if err := tx.Create(&roles).Error; err != nil {
		tx.Rollback()
		logger.Errorf("bootstrap fail: create roles: %v", err)
		return
	}

	admin := model.SysUser{
		TenantBase: model.TenantBase{
			Base:      base,
			CompanyID: company.ID,
		},
		StoreID:  store.ID,
		Username: "admin",
		Password: string(hash),
		Nickname: "超级管理员",
		RoleID:   roles[0].ID,
		Status:   1,
	}
	if err := tx.Create(&admin).Error; err != nil {
		tx.Rollback()
		logger.Errorf("bootstrap fail: create admin: %v", err)
		return
	}

	payments := []model.PaymentMethod{
		{TenantBase: model.TenantBase{Base: base, CompanyID: company.ID}, Name: "微信支付", Type: "wechat", Status: 1, Sort: 1},
		{TenantBase: model.TenantBase{Base: base, CompanyID: company.ID}, Name: "支付宝", Type: "alipay", Status: 1, Sort: 2},
		{TenantBase: model.TenantBase{Base: base, CompanyID: company.ID}, Name: "银行转账", Type: "bank", Status: 1, Sort: 3},
		{TenantBase: model.TenantBase{Base: base, CompanyID: company.ID}, Name: "现金", Type: "cash", Status: 1, Sort: 4},
	}
	if err := tx.Create(&payments).Error; err != nil {
		tx.Rollback()
		logger.Errorf("bootstrap fail: create payment methods: %v", err)
		return
	}

	if err := tx.Commit().Error; err != nil {
		logger.Errorf("bootstrap fail: commit: %v", err)
		return
	}
	logger.Infof("bootstrap done: company=%d store=%d roles=%d admin=admin/admin123456", company.ID, store.ID, len(roles))
}
