package service

import (
	"photography-server/internal/common"
	"photography-server/internal/enum"
	"photography-server/internal/model"
	"photography-server/internal/pkg/errs"
	"photography-server/internal/presentation/dto"
)

func (s *Service) ListCalendar(op Operator, startDate, endDate string, photographerID int64) ([]model.CalendarBlock, error) {
	return s.CalendarRepo.List(op.CompanyID, startDate, endDate, photographerID)
}

func (s *Service) BlockCalendar(op Operator, req dto.CalendarBlockReq) (*model.CalendarBlock, error) {
	block := model.CalendarBlock{
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
		Remark:         req.Remark,
		Status:         enum.BlockStatusLocked,
	}
	if err := s.DB().Create(&block).Error; err != nil {
		return nil, err
	}
	return &block, nil
}

func (s *Service) CancelCalendarBlock(op Operator, id int64) error {
	_, err := s.CalendarRepo.GetByID(op.CompanyID, id)
	if err != nil {
		return errs.NotFound(common.ErrCalendarNotFound)
	}
	return s.CalendarRepo.Update(op.CompanyID, id, map[string]interface{}{
		"status": enum.BlockStatusCancelled, "updated_by": op.UserID,
	})
}
