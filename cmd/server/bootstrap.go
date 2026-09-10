package main

import (
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"photography-server/internal/model"
	"photography-server/internal/pkg/logger"
)

// bootstrap 首次启动时初始化默认租户数据（公司/门店/角色/超级管理员）。
// 仅限非生产环境（dev/test/docker.dev）：生产初始化必须走 docs/sql/dml.sql 手工导入，
// 禁止在库空时自动创建固定口令超管（#25）。
// 默认账号 admin；非生产默认密码 admin123456（仅本地/测试联调，登录后请尽快改密）。
// db 由组合根注入（#40）。
func bootstrap(profile string, db *gorm.DB) {
	// #25：prod 一律不做自动 bootstrap——库空时自动建号 = 固定口令超管直接上线，
	// 即便库已有数据，初始化也不应在生产重复出现；生产初始化统一走受控的 dml.sql。
	if profile == "prod" {
		logger.Warnf("bootstrap disabled in prod: 请使用 docs/sql/dml.sql 手工初始化租户与超管，禁止自动创建固定口令账号")
		return
	}

	var companyCount int64
	if err := db.Model(&model.SysCompany{}).Count(&companyCount).Error; err != nil {
		// #25：Count 出错必须中止，不能在数据库异常状态下写入初始化数据
		logger.Errorf("bootstrap abort: 检查公司数据失败: %v", err)
		return
	}
	if companyCount > 0 {
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
	// #25：日志不打印口令；默认口令仅限非生产（prod 已在函数入口禁用自动 bootstrap）
	logger.Infof("bootstrap done: company=%d store=%d roles=%d admin username=admin（默认口令仅限非生产，请登录后立即修改）", company.ID, store.ID, len(roles))
}
