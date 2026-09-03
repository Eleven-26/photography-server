package repository

import (
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

func (r *CalendarRepo) List(companyID int64, startDate, endDate string, photographerID int64) ([]model.CalendarBlock, error) {
	q := r.tenant(companyID)
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

func (r *CalendarRepo) GetByID(companyID, blockID int64) (*model.CalendarBlock, error) {
	var b model.CalendarBlock
	if err := r.tenant(companyID).First(&b, blockID).Error; err != nil {
		return nil, err
	}
	return &b, nil
}

// Create 创建档期锁（company_id 由调用方在 model 上填充）
func (r *CalendarRepo) Create(b *model.CalendarBlock) error {
	return r.conn().Create(b).Error
}

func (r *CalendarRepo) Update(companyID, blockID int64, updates map[string]interface{}) error {
	return r.tenant(companyID).Model(&model.CalendarBlock{}).Where("id = ?", blockID).Updates(updates).Error
}

func (r *CalendarRepo) CountByPhotographer(companyID int64, photographerID int64, date string) (int64, error) {
	var count int64
	err := r.tenant(companyID).Model(&model.CalendarBlock{}).
		Where("photographer_id = ? AND date = ?", photographerID, date).
		Count(&count).Error
	return count, err
}
