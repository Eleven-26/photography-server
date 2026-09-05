package repository

import (
	"context"

	"gorm.io/gorm"
	"photography-server/internal/model"
)

type CalendarRepo struct {
	Repo
}

// WithTx 返回绑定到指定事务连接的副本，事务内的所有写操作将复用该连接，
// 保证跨多张表的写入原子性（失败自动回滚）。
func (r *CalendarRepo) WithTx(tx *gorm.DB) *CalendarRepo {
	return &CalendarRepo{Repo: Repo{db: tx}}
}

func NewCalendarRepo() *CalendarRepo { return &CalendarRepo{} }

func (r *CalendarRepo) List(ctx context.Context, companyID int64, startDate, endDate string, photographerID int64) ([]model.CalendarBlock, error) {
	q := r.tenant(companyID).WithContext(ctx)
	if startDate != "" && endDate != "" {
		q = q.Where("date BETWEEN ? AND ?", startDate, endDate)
	}
	if photographerID > 0 {
		q = q.Where("photographer_id = ?", photographerID)
	}
	var list []model.CalendarBlock
	err := q.Order("date ASC, time_range ASC").Find(&list).Error
	return list, err
}

func (r *CalendarRepo) GetByID(ctx context.Context, companyID, blockID int64) (*model.CalendarBlock, error) {
	var b model.CalendarBlock
	if err := r.tenant(companyID).WithContext(ctx).First(&b, blockID).Error; err != nil {
		return nil, err
	}
	return &b, nil
}

// Create 创建档期锁（company_id 由调用方在 model 上填充）
func (r *CalendarRepo) Create(ctx context.Context, b *model.CalendarBlock) error {
	return r.conn().WithContext(ctx).Create(b).Error
}

func (r *CalendarRepo) Update(ctx context.Context, companyID, blockID int64, updates map[string]interface{}) error {
	return r.tenant(companyID).WithContext(ctx).Model(&model.CalendarBlock{}).Where("id = ?", blockID).Updates(updates).Error
}

func (r *CalendarRepo) CountByPhotographer(ctx context.Context, companyID int64, photographerID int64, date string) (int64, error) {
	var count int64
	err := r.tenant(companyID).WithContext(ctx).Model(&model.CalendarBlock{}).
		Where("photographer_id = ? AND date = ?", photographerID, date).
		Count(&count).Error
	return count, err
}
