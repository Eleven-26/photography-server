package repository

import (
	"photography-server/internal/model"
)

type CalendarRepo struct{}

func NewCalendarRepo() *CalendarRepo { return &CalendarRepo{} }

func (r *CalendarRepo) List(companyID int64, startDate, endDate string, photographerID int64) ([]model.CalendarBlock, error) {
	q := tenant(companyID)
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
	if err := tenant(companyID).First(&b, blockID).Error; err != nil {
		return nil, err
	}
	return &b, nil
}

func (r *CalendarRepo) Update(companyID, blockID int64, updates map[string]interface{}) error {
	return tenant(companyID).Model(&model.CalendarBlock{}).Where("id = ?", blockID).Updates(updates).Error
}

func (r *CalendarRepo) CountByPhotographer(companyID int64, photographerID int64, date string) (int64, error) {
	var count int64
	err := tenant(companyID).Model(&model.CalendarBlock{}).
		Where("photographer_id = ? AND date = ?", photographerID, date).
		Count(&count).Error
	return count, err
}
