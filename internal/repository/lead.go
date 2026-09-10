package repository

import (
	"context"

	"gorm.io/gorm"

	"photography-server/internal/enum"
	"photography-server/internal/model"
)

type LeadRepo struct {
	Repo
}

// WithTx 返回绑定到指定事务连接的副本，事务内的所有写操作将复用该连接，
// 保证跨多张表的写入原子性（失败自动回滚）。
func (r *LeadRepo) WithTx(tx *gorm.DB) *LeadRepo {
	return &LeadRepo{Repo: Repo{db: tx}}
}

func NewLeadRepo() *LeadRepo {
	return &LeadRepo{}
}

// List 线索列表（分页 + 关键字 + 状态 + 负责人筛选）
func (r *LeadRepo) List(ctx context.Context, companyID int64, page, pageSize int, keyword, status string, ownerID int64) ([]model.Lead, int64, error) {
	q := r.tenant(companyID).WithContext(ctx)
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
func (r *LeadRepo) GetByID(ctx context.Context, companyID, id int64) (*model.Lead, error) {
	var l model.Lead
	if err := r.tenant(companyID).WithContext(ctx).First(&l, id).Error; err != nil {
		return nil, err
	}
	return &l, nil
}

// Create 创建线索
func (r *LeadRepo) Create(ctx context.Context, l *model.Lead) error {
	return r.tenant(l.CompanyID).WithContext(ctx).Create(l).Error
}

// Update 更新线索
func (r *LeadRepo) Update(ctx context.Context, companyID, id int64, updates map[string]interface{}) error {
	return r.tenant(companyID).WithContext(ctx).Model(&model.Lead{}).Where("id = ?", id).Updates(updates).Error
}

// Delete 删除线索
func (r *LeadRepo) Delete(ctx context.Context, companyID, id int64) error {
	return r.tenant(companyID).WithContext(ctx).Delete(&model.Lead{}, id).Error
}

// GetByMobile 根据手机号查询线索
func (r *LeadRepo) GetByMobile(ctx context.Context, companyID int64, mobile string) (*model.Lead, error) {
	var l model.Lead
	if err := r.tenant(companyID).WithContext(ctx).Where("mobile = ?", mobile).First(&l).Error; err != nil {
		return nil, err
	}
	return &l, nil
}

// ListFollowUpDue 今日待跟进线索：下次跟进时间已到（含逾期）且尚未成交/流失的线索。
// until 传「今日 23:59:59」，让当天到期的线索也进入待办；按到期时间升序——越早到期越优先。
// 注：next_follow_at 为 datetime 列，NULL 不参与比较（SQL 三值逻辑），
// 因此只需 IS NOT NULL，不能写 `!= ''`——严格模式下空串转 datetime 会直接报错。
func (r *LeadRepo) ListFollowUpDue(ctx context.Context, companyID int64, until string, limit int) ([]model.Lead, error) {
	q := r.tenant(companyID).WithContext(ctx).
		Where("next_follow_at IS NOT NULL AND next_follow_at <= ?", until).
		Where("status IN ?", []int{
			int(enum.LeadStatusPending),
			int(enum.LeadStatusQuoting),
			int(enum.LeadStatusQuoted),
		}).
		Order("next_follow_at ASC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	var list []model.Lead
	err := q.Find(&list).Error
	return list, err
}

// -------- 报价单 --------

// GetQuoteByID 根据 ID 查询报价单
func (r *LeadRepo) GetQuoteByID(ctx context.Context, companyID, id int64) (*model.Quote, error) {
	var q model.Quote
	if err := r.tenant(companyID).WithContext(ctx).First(&q, id).Error; err != nil {
		return nil, err
	}
	return &q, nil
}

// CreateQuote 创建报价单
func (r *LeadRepo) CreateQuote(ctx context.Context, q *model.Quote) error {
	return r.tenant(q.CompanyID).WithContext(ctx).Create(q).Error
}

// ListQuotesByLead 查询线索下的报价单列表
func (r *LeadRepo) ListQuotesByLead(ctx context.Context, companyID, leadID int64) ([]model.Quote, error) {
	var list []model.Quote
	err := r.tenant(companyID).WithContext(ctx).Where("lead_id = ?", leadID).Order("id DESC").Find(&list).Error
	return list, err
}

// ListQuotesByCustomer 查询客户可见的报价单（H5「我的报价」）。
// 两条来源：直接挂在客户名下的报价（quote.customer_id，报价创建时由 lead.customer_id 回填），
// 以及挂在该客户线索上的报价——早期数据可能未回填 customer_id，只按前者查会漏，故用 lead_id 子查询兜底。
func (r *LeadRepo) ListQuotesByCustomer(ctx context.Context, companyID, customerID int64) ([]model.Quote, error) {
	leadIDs := r.conn().Model(&model.Lead{}).Select("id").
		Where("company_id = ? AND customer_id = ?", companyID, customerID)
	var list []model.Quote
	err := r.tenant(companyID).WithContext(ctx).
		Where("customer_id = ? OR lead_id IN (?)", customerID, leadIDs).
		Order("id DESC").Find(&list).Error
	return list, err
}

// UpdateQuote 更新报价单
func (r *LeadRepo) UpdateQuote(ctx context.Context, companyID, id int64, updates map[string]interface{}) error {
	return r.tenant(companyID).WithContext(ctx).Model(&model.Quote{}).Where("id = ?", id).Updates(updates).Error
}
