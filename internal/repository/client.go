package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"photography-server/internal/model"
)

// ---------------------------------------------------------------------
// 三端（小程序/APP/H5）新增实体的数据访问层
// ---------------------------------------------------------------------

// OrderRescheduleRepo 改期单
type OrderRescheduleRepo struct{ Repo }

func (r *OrderRescheduleRepo) WithTx(tx *gorm.DB) *OrderRescheduleRepo {
	return &OrderRescheduleRepo{Repo: Repo{db: tx}}
}

func NewOrderRescheduleRepo() *OrderRescheduleRepo { return &OrderRescheduleRepo{} }

func (r *OrderRescheduleRepo) Create(ctx context.Context, m *model.OrderReschedule) error {
	return r.conn().WithContext(ctx).Create(m).Error
}

func (r *OrderRescheduleRepo) GetByID(ctx context.Context, companyID, id int64) (*model.OrderReschedule, error) {
	var m model.OrderReschedule
	if err := r.tenant(companyID).WithContext(ctx).First(&m, id).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

// GetPendingByOrder 取订单当前待确认的改期单（同一订单同时最多一张）
func (r *OrderRescheduleRepo) GetPendingByOrder(ctx context.Context, companyID, orderID int64) (*model.OrderReschedule, error) {
	var m model.OrderReschedule
	err := r.tenant(companyID).WithContext(ctx).
		Where("order_id = ? AND status = 1", orderID).First(&m).Error
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *OrderRescheduleRepo) ListByOrder(ctx context.Context, companyID, orderID int64) ([]model.OrderReschedule, error) {
	var list []model.OrderReschedule
	err := r.tenant(companyID).WithContext(ctx).
		Where("order_id = ?", orderID).Order("id DESC").Find(&list).Error
	return list, err
}

func (r *OrderRescheduleRepo) List(ctx context.Context, companyID int64, page, pageSize int, status int) ([]model.OrderReschedule, int64, error) {
	q := r.tenant(companyID).WithContext(ctx)
	if status > 0 {
		q = q.Where("status = ?", status)
	}
	var total int64
	if err := q.Model(&model.OrderReschedule{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.OrderReschedule
	page, pageSize = normalizePage(page, pageSize)
	if err := q.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func (r *OrderRescheduleRepo) Update(ctx context.Context, companyID, id int64, updates map[string]interface{}) error {
	return r.tenant(companyID).WithContext(ctx).Model(&model.OrderReschedule{}).
		Where("id = ?", id).Updates(updates).Error
}

// ReviewRepo 订单评价
type ReviewRepo struct{ Repo }

func (r *ReviewRepo) WithTx(tx *gorm.DB) *ReviewRepo {
	return &ReviewRepo{Repo: Repo{db: tx}}
}

func NewReviewRepo() *ReviewRepo { return &ReviewRepo{} }

func (r *ReviewRepo) Create(ctx context.Context, m *model.OrderReview) error {
	return r.conn().WithContext(ctx).Create(m).Error
}

func (r *ReviewRepo) GetByOrderID(ctx context.Context, companyID, orderID int64) (*model.OrderReview, error) {
	var m model.OrderReview
	if err := r.tenant(companyID).WithContext(ctx).
		Where("order_id = ?", orderID).First(&m).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *ReviewRepo) List(ctx context.Context, companyID int64, page, pageSize int, minRating int) ([]model.OrderReview, int64, error) {
	q := r.tenant(companyID).WithContext(ctx)
	if minRating > 0 {
		q = q.Where("rating >= ?", minRating)
	}
	var total int64
	if err := q.Model(&model.OrderReview{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.OrderReview
	page, pageSize = normalizePage(page, pageSize)
	if err := q.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func (r *ReviewRepo) Update(ctx context.Context, companyID, id int64, updates map[string]interface{}) error {
	return r.tenant(companyID).WithContext(ctx).Model(&model.OrderReview{}).
		Where("id = ?", id).Updates(updates).Error
}

// CustomRequestRepo 定制需求
type CustomRequestRepo struct{ Repo }

func (r *CustomRequestRepo) WithTx(tx *gorm.DB) *CustomRequestRepo {
	return &CustomRequestRepo{Repo: Repo{db: tx}}
}

func NewCustomRequestRepo() *CustomRequestRepo { return &CustomRequestRepo{} }

func (r *CustomRequestRepo) Create(ctx context.Context, m *model.CustomRequest) error {
	return r.conn().WithContext(ctx).Create(m).Error
}

func (r *CustomRequestRepo) GetByID(ctx context.Context, companyID, id int64) (*model.CustomRequest, error) {
	var m model.CustomRequest
	if err := r.tenant(companyID).WithContext(ctx).First(&m, id).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *CustomRequestRepo) List(ctx context.Context, companyID int64, page, pageSize int, status int, customerID int64) ([]model.CustomRequest, int64, error) {
	q := r.tenant(companyID).WithContext(ctx)
	if status > 0 {
		q = q.Where("status = ?", status)
	}
	if customerID > 0 {
		q = q.Where("customer_id = ?", customerID)
	}
	var total int64
	if err := q.Model(&model.CustomRequest{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.CustomRequest
	page, pageSize = normalizePage(page, pageSize)
	if err := q.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func (r *CustomRequestRepo) Update(ctx context.Context, companyID, id int64, updates map[string]interface{}) error {
	return r.tenant(companyID).WithContext(ctx).Model(&model.CustomRequest{}).
		Where("id = ?", id).Updates(updates).Error
}

// AddonRepo 订单加项
type AddonRepo struct{ Repo }

func (r *AddonRepo) WithTx(tx *gorm.DB) *AddonRepo {
	return &AddonRepo{Repo: Repo{db: tx}}
}

func NewAddonRepo() *AddonRepo { return &AddonRepo{} }

func (r *AddonRepo) CreateBatch(ctx context.Context, list []model.OrderAddon) error {
	if len(list) == 0 {
		return nil
	}
	return r.conn().WithContext(ctx).Create(&list).Error
}

func (r *AddonRepo) ListByOrder(ctx context.Context, companyID, orderID int64) ([]model.OrderAddon, error) {
	var list []model.OrderAddon
	err := r.tenant(companyID).WithContext(ctx).
		Where("order_id = ?", orderID).Order("id ASC").Find(&list).Error
	return list, err
}

func (r *AddonRepo) GetByID(ctx context.Context, companyID, id int64) (*model.OrderAddon, error) {
	var m model.OrderAddon
	if err := r.tenant(companyID).WithContext(ctx).Where("id = ?", id).First(&m).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *AddonRepo) Delete(ctx context.Context, companyID, id int64) error {
	return r.tenant(companyID).WithContext(ctx).
		Where("id = ?", id).Delete(&model.OrderAddon{}).Error
}

func (r *AddonRepo) Update(ctx context.Context, companyID, id int64, updates map[string]interface{}) error {
	return r.tenant(companyID).WithContext(ctx).Model(&model.OrderAddon{}).
		Where("id = ?", id).Updates(updates).Error
}

func (r *AddonRepo) DeleteByOrder(ctx context.Context, companyID, orderID int64) error {
	return r.tenant(companyID).WithContext(ctx).
		Where("order_id = ?", orderID).Delete(&model.OrderAddon{}).Error
}

// LeadExtraRepo 线索沟通记录 + AI 简报项
type LeadExtraRepo struct{ Repo }

func (r *LeadExtraRepo) WithTx(tx *gorm.DB) *LeadExtraRepo {
	return &LeadExtraRepo{Repo: Repo{db: tx}}
}

func NewLeadExtraRepo() *LeadExtraRepo { return &LeadExtraRepo{} }

func (r *LeadExtraRepo) CreateMessage(ctx context.Context, m *model.LeadMessage) error {
	return r.conn().WithContext(ctx).Create(m).Error
}

func (r *LeadExtraRepo) ListMessages(ctx context.Context, companyID, leadID int64) ([]model.LeadMessage, error) {
	var list []model.LeadMessage
	err := r.tenant(companyID).WithContext(ctx).
		Where("lead_id = ?", leadID).Order("id ASC").Find(&list).Error
	return list, err
}

func (r *LeadExtraRepo) CreateBriefItem(ctx context.Context, m *model.LeadBriefItem) error {
	return r.conn().WithContext(ctx).Create(m).Error
}

func (r *LeadExtraRepo) CreateBriefItems(ctx context.Context, list []model.LeadBriefItem) error {
	if len(list) == 0 {
		return nil
	}
	return r.conn().WithContext(ctx).Create(&list).Error
}

func (r *LeadExtraRepo) ListBriefItems(ctx context.Context, companyID, leadID int64) ([]model.LeadBriefItem, error) {
	var list []model.LeadBriefItem
	err := r.tenant(companyID).WithContext(ctx).
		Where("lead_id = ?", leadID).Order("affects_pricing DESC, sort ASC, id ASC").Find(&list).Error
	return list, err
}

func (r *LeadExtraRepo) GetBriefItem(ctx context.Context, companyID, id int64) (*model.LeadBriefItem, error) {
	var m model.LeadBriefItem
	if err := r.tenant(companyID).WithContext(ctx).First(&m, id).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *LeadExtraRepo) UpdateBriefItem(ctx context.Context, companyID, id int64, updates map[string]interface{}) error {
	return r.tenant(companyID).WithContext(ctx).Model(&model.LeadBriefItem{}).
		Where("id = ?", id).Updates(updates).Error
}

func (r *LeadExtraRepo) DeleteBriefItemsByLead(ctx context.Context, companyID, leadID int64) error {
	return r.tenant(companyID).WithContext(ctx).
		Where("lead_id = ?", leadID).Delete(&model.LeadBriefItem{}).Error
}

// SlotTemplateRepo 档期时段模板
type SlotTemplateRepo struct{ Repo }

func (r *SlotTemplateRepo) WithTx(tx *gorm.DB) *SlotTemplateRepo {
	return &SlotTemplateRepo{Repo: Repo{db: tx}}
}

func NewSlotTemplateRepo() *SlotTemplateRepo { return &SlotTemplateRepo{} }

func (r *SlotTemplateRepo) Create(ctx context.Context, m *model.SlotTemplate) error {
	return r.conn().WithContext(ctx).Create(m).Error
}

func (r *SlotTemplateRepo) List(ctx context.Context, companyID, photographerID int64) ([]model.SlotTemplate, error) {
	q := r.tenant(companyID).WithContext(ctx).Where("status = 1")
	if photographerID > 0 {
		q = q.Where("photographer_id IN ?", []int64{0, photographerID})
	}
	var list []model.SlotTemplate
	err := q.Order("weekday ASC, start_time ASC").Find(&list).Error
	return list, err
}

func (r *SlotTemplateRepo) GetByID(ctx context.Context, companyID, id int64) (*model.SlotTemplate, error) {
	var m model.SlotTemplate
	if err := r.tenant(companyID).WithContext(ctx).First(&m, id).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *SlotTemplateRepo) Update(ctx context.Context, companyID, id int64, updates map[string]interface{}) error {
	return r.tenant(companyID).WithContext(ctx).Model(&model.SlotTemplate{}).
		Where("id = ?", id).Updates(updates).Error
}

func (r *SlotTemplateRepo) Delete(ctx context.Context, companyID, id int64) error {
	return r.tenant(companyID).WithContext(ctx).Delete(&model.SlotTemplate{}, id).Error
}

// StudioSettingRepo 工作室设置（company 唯一，不存在时由 GetOrCreate 兜底）
type StudioSettingRepo struct{ Repo }

func (r *StudioSettingRepo) WithTx(tx *gorm.DB) *StudioSettingRepo {
	return &StudioSettingRepo{Repo: Repo{db: tx}}
}

func NewStudioSettingRepo() *StudioSettingRepo { return &StudioSettingRepo{} }

func (r *StudioSettingRepo) GetByCompany(ctx context.Context, companyID int64) (*model.StudioSetting, error) {
	var m model.StudioSetting
	err := r.tenant(companyID).WithContext(ctx).First(&m).Error
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// GetBySlug 按预约主页短链标识反查工作室设置（跨租户全局查询，不经 tenant 过滤）。
// slug 是面向客户公开的不可枚举标识（预约主页 URL/二维码），服务端反查 company_id，
// 替代"客户端直接传 company_id"的裸租户定位（审查报告 #29）。
func (r *StudioSettingRepo) GetBySlug(ctx context.Context, slug string) (*model.StudioSetting, error) {
	var m model.StudioSetting
	if err := r.conn().WithContext(ctx).Where("homepage_slug = ?", slug).Order("id ASC").First(&m).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

// GetOrCreate 取工作室设置，不存在则创建默认配置（保证三端读取总有值）
func (r *StudioSettingRepo) GetOrCreate(ctx context.Context, companyID int64) (*model.StudioSetting, error) {
	m, err := r.GetByCompany(ctx, companyID)
	if err == nil {
		return m, nil
	}
	m = &model.StudioSetting{
		Base:      model.Base{CreatedAt: time.Now(), UpdatedAt: time.Now()},
		CompanyID: companyID,
	}
	if err := r.conn().WithContext(ctx).Create(m).Error; err != nil {
		// 并发下可能唯一键冲突，重试读取
		return r.GetByCompany(ctx, companyID)
	}
	return m, nil
}

func (r *StudioSettingRepo) Update(ctx context.Context, companyID int64, updates map[string]interface{}) error {
	return r.tenant(companyID).WithContext(ctx).Model(&model.StudioSetting{}).
		Where("company_id = ?", companyID).Updates(updates).Error
}

// DeviceRepo 登录设备
type DeviceRepo struct{ Repo }

func (r *DeviceRepo) WithTx(tx *gorm.DB) *DeviceRepo {
	return &DeviceRepo{Repo: Repo{db: tx}}
}

func NewDeviceRepo() *DeviceRepo { return &DeviceRepo{} }

func (r *DeviceRepo) Create(ctx context.Context, m *model.UserDevice) error {
	return r.conn().WithContext(ctx).Create(m).Error
}

func (r *DeviceRepo) ListByUser(ctx context.Context, companyID, userID int64) ([]model.UserDevice, error) {
	var list []model.UserDevice
	err := r.tenant(companyID).WithContext(ctx).
		Where("user_id = ? AND status = 1", userID).Order("last_active_at DESC, id DESC").Find(&list).Error
	return list, err
}

func (r *DeviceRepo) Update(ctx context.Context, companyID, id int64, updates map[string]interface{}) error {
	return r.tenant(companyID).WithContext(ctx).Model(&model.UserDevice{}).
		Where("id = ?", id).Updates(updates).Error
}

func (r *DeviceRepo) Delete(ctx context.Context, companyID, id int64) error {
	return r.tenant(companyID).WithContext(ctx).Delete(&model.UserDevice{}, id).Error
}

// GetByMobile 按手机号查客户（客户端验证码登录：存在即登录，不存在可自动注册）
func (r *CustomerRepo) GetByMobile(ctx context.Context, companyID int64, mobile string) (*model.Customer, error) {
	var c model.Customer
	err := r.tenant(companyID).WithContext(ctx).
		Where("mobile = ?", mobile).First(&c).Error
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// GetByOpenID 按小程序 openid 查客户
func (r *CustomerRepo) GetByOpenID(ctx context.Context, companyID int64, openid string) (*model.Customer, error) {
	var c model.Customer
	err := r.tenant(companyID).WithContext(ctx).
		Where("openid = ?", openid).First(&c).Error
	if err != nil {
		return nil, err
	}
	return &c, nil
}
