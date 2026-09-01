package repository

import (
	"photography-server/internal/model"
)

type LeadRepo struct{}

func NewLeadRepo() *LeadRepo {
	return &LeadRepo{}
}

// List 线索列表（分页 + 关键字 + 状态 + 负责人筛选）
func (r *LeadRepo) List(companyID int64, page, pageSize int, keyword, status string, ownerID int64) ([]model.Lead, int64, error) {
	q := tenant(companyID)
	if keyword != "" {
		kw := "%" + keyword + "%"
		q = q.Where("name LIKE ? OR mobile LIKE ? OR code LIKE ?", kw, kw, kw)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if ownerID > 0 {
		q = q.Where("owner_id = ?", ownerID)
	}
	var total int64
	if err := q.Model(&model.Lead{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.Lead
	page, pageSize = normalizePage(page, pageSize)
	if err := q.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// GetByID 根据 ID 查询线索
func (r *LeadRepo) GetByID(companyID, id int64) (*model.Lead, error) {
	var l model.Lead
	if err := tenant(companyID).First(&l, id).Error; err != nil {
		return nil, err
	}
	return &l, nil
}

// Create 创建线索
func (r *LeadRepo) Create(l *model.Lead) error {
	return tenant(l.CompanyID).Create(l).Error
}

// Update 更新线索
func (r *LeadRepo) Update(companyID, id int64, updates map[string]interface{}) error {
	return tenant(companyID).Model(&model.Lead{}).Where("id = ?", id).Updates(updates).Error
}

// Delete 删除线索
func (r *LeadRepo) Delete(companyID, id int64) error {
	return tenant(companyID).Delete(&model.Lead{}, id).Error
}

// GetByMobile 根据手机号查询线索
func (r *LeadRepo) GetByMobile(companyID int64, mobile string) (*model.Lead, error) {
	var l model.Lead
	if err := tenant(companyID).Where("mobile = ?", mobile).First(&l).Error; err != nil {
		return nil, err
	}
	return &l, nil
}

// -------- 报价单 --------

// GetQuoteByID 根据 ID 查询报价单
func (r *LeadRepo) GetQuoteByID(companyID, id int64) (*model.Quote, error) {
	var q model.Quote
	if err := tenant(companyID).First(&q, id).Error; err != nil {
		return nil, err
	}
	return &q, nil
}

// CreateQuote 创建报价单
func (r *LeadRepo) CreateQuote(q *model.Quote) error {
	return tenant(q.CompanyID).Create(q).Error
}

// ListQuotesByLead 查询线索下的报价单列表
func (r *LeadRepo) ListQuotesByLead(companyID, leadID int64) ([]model.Quote, error) {
	var list []model.Quote
	err := tenant(companyID).Where("lead_id = ?", leadID).Order("id DESC").Find(&list).Error
	return list, err
}

// UpdateQuote 更新报价单
func (r *LeadRepo) UpdateQuote(companyID, id int64, updates map[string]interface{}) error {
	return tenant(companyID).Model(&model.Quote{}).Where("id = ?", id).Updates(updates).Error
}
