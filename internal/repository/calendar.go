package repository

import (
	"photography-server/internal/model"
)

type CalendarRepo struct{}

func NewCalendarRepo() *CalendarRepo {
	return &CalendarRepo{}
}

// List 档期列表（按日期范围筛选）
func (r *CalendarRepo) List(companyID int64, startDate, endDate string) ([]model.CalendarBlock, error) {
	q := tenant(companyID)
	if startDate != "" {
		q = q.Where("date >= ?", startDate)
	}
	if endDate != "" {
		q = q.Where("date <= ?", endDate)
	}
	var list []model.CalendarBlock
	err := q.Order("date ASC, time_range ASC").Find(&list).Error
	return list, err
}

// CountByPhotographer 统计摄影师在指定时间段的锁定档期数
func (r *CalendarRepo) CountByPhotographer(companyID int64, date, timeRange string, photographerID int64) (int64, error) {
	var count int64
	err := tenant(companyID).Model(&model.CalendarBlock{}).
		Where("date = ? AND time_range = ? AND photographer_id = ? AND status = ?",
			date, timeRange, photographerID, model.BlockStatusLocked).
		Count(&count).Error
	return count, err
}

// Create 创建档期
func (r *CalendarRepo) Create(b *model.CalendarBlock) error {
	return tenant(b.CompanyID).Create(b).Error
}

// GetByID 根据 ID 查询档期
func (r *CalendarRepo) GetByID(companyID, id int64) (*model.CalendarBlock, error) {
	var b model.CalendarBlock
	if err := tenant(companyID).First(&b, id).Error; err != nil {
		return nil, err
	}
	return &b, nil
}

// Update 更新档期状态
func (r *CalendarRepo) Update(companyID, id int64, updates map[string]interface{}) error {
	return tenant(companyID).Model(&model.CalendarBlock{}).Where("id = ?", id).Updates(updates).Error
}
