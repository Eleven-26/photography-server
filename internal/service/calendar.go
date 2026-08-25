package service

import (
	"photography-server/internal/model"
	"photography-server/internal/pkg/errs"
)

type BlockReq struct {
	StoreID        int64  `json:"store_id"`
	OrderID        int64  `json:"order_id"`
	CustomerID     int64  `json:"customer_id"`
	CustomerName   string `json:"customer_name"`
	Date           string `json:"date" binding:"required"`
	TimeRange      string `json:"time_range" binding:"required"`
	ProjectType    string `json:"project_type"`
	PhotographerID int64  `json:"photographer_id"`
	Photographer   string `json:"photographer"`
	Remark         string `json:"remark"`
}

func (s *Service) ListCalendar(op Operator, startDate, endDate string) ([]model.CalendarBlock, error) {
	q := s.tenant(op)
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

// LockBlock 锁定档期：校验同一摄影师同一时间段是否已占用
func (s *Service) LockBlock(op Operator, req BlockReq) error {
	var count int64
	s.tenant(op).Model(&model.CalendarBlock{}).
		Where("date = ? AND time_range = ? AND photographer_id = ? AND status = ?",
			req.Date, req.TimeRange, req.PhotographerID, model.BlockStatusLocked).
		Count(&count)
	if count > 0 {
		return errs.Conflict("该档期已被锁定，请选择其他时间段")
	}
	b := model.CalendarBlock{
		TenantBase: model.TenantBase{
			Base:      model.Base{CreatedBy: op.UserID, UpdatedBy: op.UserID},
			CompanyID: op.CompanyID,
		},
		StoreID:        orDefaultInt64(req.StoreID, op.StoreID),
		OrderID:        req.OrderID,
		CustomerID:     req.CustomerID,
		CustomerName:   req.CustomerName,
		Date:           req.Date,
		TimeRange:      req.TimeRange,
		ProjectType:    req.ProjectType,
		PhotographerID: req.PhotographerID,
		Photographer:   req.Photographer,
		Status:         model.BlockStatusLocked,
		Remark:         req.Remark,
	}
	return s.tenant(op).Create(&b).Error
}

func (s *Service) CancelBlock(op Operator, id int64) error {
	var b model.CalendarBlock
	if err := s.tenant(op).First(&b, id).Error; err != nil {
		return errs.NotFound("档期不存在")
	}
	return s.tenant(op).Model(&b).Updates(map[string]interface{}{
		"status": model.BlockStatusCancelled, "updated_by": op.UserID,
	}).Error
}
