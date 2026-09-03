package repository

import (
	"gorm.io/gorm"
	"photography-server/internal/model"
)

type SettingsRepo struct {
	Repo
}

// WithTx 返回绑定到指定事务连接的副本，事务内的所有写操作将复用该连接，
// 保证跨多张表的写入原子性（失败自动回滚）。
func (r *SettingsRepo) WithTx(tx *gorm.DB) *SettingsRepo {
	return &SettingsRepo{Repo: Repo{db: tx}}
}

func NewSettingsRepo() *SettingsRepo {
	return &SettingsRepo{}
}

// GetCompany 获取公司信息
func (r *SettingsRepo) GetCompany(companyID int64) (*model.SysCompany, error) {
	var c model.SysCompany
	if err := r.conn().First(&c, companyID).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

// UpdateCompany 更新公司信息
func (r *SettingsRepo) UpdateCompany(companyID int64, updates map[string]interface{}) error {
	return r.conn().Model(&model.SysCompany{}).Where("id = ?", companyID).Updates(updates).Error
}

// ListStores 门店列表
func (r *SettingsRepo) ListStores(companyID int64) ([]model.SysStore, error) {
	var list []model.SysStore
	err := r.tenant(companyID).Order("id ASC").Find(&list).Error
	return list, err
}

// ListRoles 角色列表
func (r *SettingsRepo) ListRoles(companyID int64) ([]model.SysRole, error) {
	var list []model.SysRole
	err := r.tenant(companyID).Order("id ASC").Find(&list).Error
	return list, err
}

// ListPaymentMethods 收款方式列表
func (r *SettingsRepo) ListPaymentMethods(companyID int64) ([]model.PaymentMethod, error) {
	var list []model.PaymentMethod
	err := r.tenant(companyID).Order("sort ASC, id ASC").Find(&list).Error
	return list, err
}

// GetPaymentMethodByID 根据 ID 查询收款方式
func (r *SettingsRepo) GetPaymentMethodByID(companyID, id int64) (*model.PaymentMethod, error) {
	var m model.PaymentMethod
	if err := r.tenant(companyID).First(&m, id).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

// CreatePaymentMethod 创建收款方式
func (r *SettingsRepo) CreatePaymentMethod(m *model.PaymentMethod) error {
	return r.tenant(m.CompanyID).Create(m).Error
}

// UpdatePaymentMethod 更新收款方式
func (r *SettingsRepo) UpdatePaymentMethod(companyID, id int64, updates map[string]interface{}) error {
	return r.tenant(companyID).Model(&model.PaymentMethod{}).Where("id = ?", id).Updates(updates).Error
}

// DeletePaymentMethod 删除收款方式
func (r *SettingsRepo) DeletePaymentMethod(companyID, id int64) error {
	return r.tenant(companyID).Delete(&model.PaymentMethod{}, id).Error
}

// ListOperationLogs 操作日志列表（分页）
func (r *SettingsRepo) ListOperationLogs(companyID int64, page, pageSize int) ([]model.SysOperationLog, int64, error) {
	q := r.tenant(companyID)
	var total int64
	if err := q.Model(&model.SysOperationLog{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.SysOperationLog
	page, pageSize = normalizePage(page, pageSize)
	if err := q.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}
