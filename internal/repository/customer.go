package repository

import (
	"context"

	"gorm.io/gorm"

	"photography-server/internal/enum"
	"photography-server/internal/model"
)

type CustomerRepo struct {
	Repo
}

// WithTx 返回绑定到指定事务连接的副本，事务内的所有写操作将复用该连接，
// 保证跨多张表的写入原子性（失败自动回滚）。
func (r *CustomerRepo) WithTx(tx *gorm.DB) *CustomerRepo {
	return &CustomerRepo{Repo: Repo{db: tx}}
}

func NewCustomerRepo() *CustomerRepo { return &CustomerRepo{} }

func (r *CustomerRepo) List(ctx context.Context, companyID int64, page, pageSize int, keyword string) ([]model.Customer, int64, error) {
	q := r.tenant(companyID).WithContext(ctx)
	if keyword != "" {
		kw := "%" + keyword + "%"
		q = q.Where("name LIKE ? OR mobile LIKE ? OR code LIKE ?", kw, kw, kw)
	}
	var total int64
	if err := q.Model(&model.Customer{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.Customer
	page, pageSize = normalizePage(page, pageSize)
	if err := q.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func (r *CustomerRepo) GetByID(ctx context.Context, companyID, customerID int64) (*model.Customer, error) {
	var c model.Customer
	if err := r.tenant(companyID).WithContext(ctx).First(&c, customerID).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *CustomerRepo) Create(ctx context.Context, c *model.Customer) error {
	return r.conn().WithContext(ctx).Create(c).Error
}

func (r *CustomerRepo) Update(ctx context.Context, companyID, customerID int64, updates map[string]interface{}) error {
	return r.tenant(companyID).WithContext(ctx).Model(&model.Customer{}).Where("id = ?", customerID).Updates(updates).Error
}

func (r *CustomerRepo) Delete(ctx context.Context, companyID, customerID int64) error {
	return r.tenant(companyID).WithContext(ctx).Delete(&model.Customer{}, customerID).Error
}

// CustomerStats 客户统计（仓储自持的领域结构，避免反向依赖 presentation/dto）
type CustomerStats struct {
	Total  int64
	Active int64
	GoldUp int64
}

// GetStats 客户统计：总数 / 活跃数 / 黄金及以上等级数
func (r *CustomerRepo) GetStats(ctx context.Context, companyID int64) (*CustomerStats, error) {
	var st CustomerStats
	q := r.tenant(companyID).WithContext(ctx)

	if err := q.Model(&model.Customer{}).Count(&st.Total).Error; err != nil {
		return nil, err
	}

	if err := q.Model(&model.Customer{}).Where("status = ?", enum.CustomerStatusActive).Count(&st.Active).Error; err != nil {
		return nil, err
	}

	if err := q.Model(&model.Customer{}).Where("level IN ?", []enum.CustomerLevel{enum.CustomerLevelGold, enum.CustomerLevelPlatinum, enum.CustomerLevelDiamond}).Count(&st.GoldUp).Error; err != nil {
		return nil, err
	}

	return &st, nil
}
