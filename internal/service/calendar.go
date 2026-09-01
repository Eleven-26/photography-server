package service

import (
	"errors"

	"gorm.io/gorm"

	"photography-server/internal/common"
	"photography-server/internal/model"
	"photography-server/internal/pkg/errs"
	"photography-server/internal/presentation/dto"
)

func (s *Service) ListCalendar(op Operator, startDate, endDate string) ([]model.CalendarBlock, error) {
	return s.CalendarRepo.List(op.CompanyID, startDate, endDate)
}

// LockBlock 锁定档期：校验同一摄影师同一时间段是否已占用
func (s *Service) LockBlock(op Operator, req dto.CalendarBlockReq) error {
	count, err := s.CalendarRepo.CountByPhotographer(op.CompanyID, req.Date, req.TimeRange, req.PhotographerID)
	if err != nil {
		return err
	}
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
	return s.CalendarRepo.Create(&b)
}

func (s *Service) CancelBlock(op Operator, id int64) error {
	_, err := s.CalendarRepo.GetByID(op.CompanyID, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errs.NotFound(common.ErrCalendarNotFound)
		}
		return err
	}
	return s.CalendarRepo.Update(op.CompanyID, id, map[string]interface{}{
		"status": model.BlockStatusCancelled, "updated_by": op.UserID,
	})
}
