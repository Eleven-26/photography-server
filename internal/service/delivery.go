package service

import (
	"photography-server/internal/domain"
	"photography-server/internal/enum"
	"photography-server/internal/model"
	"photography-server/internal/pkg/errs"
	"photography-server/internal/presentation/dto"
)

func (s *Service) CreateDelivery(op Operator, orderID int64) (*model.Delivery, error) {
	o, err := s.OrderRepo.GetByID(op.CompanyID, orderID)
	if err != nil {
		return nil, errs.NotFound(errs.ErrOrderNotFound)
	}

	d := model.Delivery{
		TenantBase: model.TenantBase{
			Base:      model.Base{CreatedBy: op.UserID, UpdatedBy: op.UserID},
			CompanyID: op.CompanyID,
		},
		Code:         domain.GenCode("DV"),
		OrderID:      orderID,
		CustomerID:   o.CustomerID,
		CustomerName: o.CustomerName,
		Stage:        enum.DeliveryStagePendingSamples,
	}
	if err := s.DeliveryRepo.Create(&d); err != nil {
		return nil, err
	}
	return &d, nil
}

func (s *Service) GetDeliveryByOrder(op Operator, orderID int64) (*model.Delivery, error) {
	return s.DeliveryRepo.GetByOrderID(op.CompanyID, orderID)
}

func (s *Service) UploadSamples(op Operator, deliveryID int64, items []dto.DeliveryItemReq) error {
	d, err := s.DeliveryRepo.GetByID(op.CompanyID, deliveryID)
	if err != nil {
		return errs.NotFound(errs.ErrDeliveryNotFound)
	}
	if d.Stage != enum.DeliveryStagePendingSamples {
		return errs.BadRequest(errs.ErrDeliveryStageInvalid)
	}

	for _, item := range items {
		di := model.DeliveryItem{
			TenantBase: model.TenantBase{
				Base:      model.Base{CreatedBy: op.UserID, UpdatedBy: op.UserID},
				CompanyID: op.CompanyID,
			},
			DeliveryID: deliveryID,
			OrderID:    d.OrderID,
			URL:        item.URL,
			FileType:   item.FileType,
			Kind:       "sample",
			Filename:   item.Filename,
			Size:       item.Size,
		}
		if err := s.DeliveryRepo.CreateItem(&di); err != nil {
			return err
		}
	}

	return s.DeliveryRepo.Update(op.CompanyID, deliveryID, map[string]interface{}{
		"stage":        enum.DeliveryStageSelecting,
		"sample_count": len(items),
	})
}

func (s *Service) SelectPhotos(op Operator, deliveryID int64, req dto.DeliverySelectReq) error {
	d, err := s.DeliveryRepo.GetByID(op.CompanyID, deliveryID)
	if err != nil {
		return errs.NotFound(errs.ErrDeliveryNotFound)
	}
	if d.Stage != enum.DeliveryStageSelecting {
		return errs.BadRequest(errs.ErrDeliveryStageInvalid)
	}

	for _, itemID := range req.ItemIDs {
		s.DeliveryRepo.UpdateItemKind(op.CompanyID, itemID, "selected")
	}

	now := "2006-01-02 15:04:05"
	return s.DeliveryRepo.Update(op.CompanyID, deliveryID, map[string]interface{}{
		"stage":          enum.DeliveryStageRetouching,
		"selected_count": len(req.ItemIDs),
		"selected_at":    now,
	})
}

func (s *Service) UploadRetouched(op Operator, deliveryID int64, items []dto.DeliveryItemReq) error {
	d, err := s.DeliveryRepo.GetByID(op.CompanyID, deliveryID)
	if err != nil {
		return errs.NotFound(errs.ErrDeliveryNotFound)
	}

	for _, item := range items {
		di := model.DeliveryItem{
			TenantBase: model.TenantBase{
				Base:      model.Base{CreatedBy: op.UserID, UpdatedBy: op.UserID},
				CompanyID: op.CompanyID,
			},
			DeliveryID: deliveryID,
			OrderID:    d.OrderID,
			URL:        item.URL,
			FileType:   item.FileType,
			Kind:       "retouched",
			Filename:   item.Filename,
			Size:       item.Size,
		}
		if err := s.DeliveryRepo.CreateItem(&di); err != nil {
			return err
		}
	}

	return s.DeliveryRepo.Update(op.CompanyID, deliveryID, map[string]interface{}{
		"stage":           enum.DeliveryStagePendingConfirm,
		"retouched_count": len(items),
	})
}

func (s *Service) ConfirmDelivered(op Operator, deliveryID int64) error {
	d, err := s.DeliveryRepo.GetByID(op.CompanyID, deliveryID)
	if err != nil {
		return errs.NotFound(errs.ErrDeliveryNotFound)
	}
	if d.Stage != enum.DeliveryStagePendingConfirm {
		return errs.BadRequest(errs.ErrDeliveryStageInvalid)
	}

	now := "2006-01-02 15:04:05"
	return s.DeliveryRepo.Update(op.CompanyID, deliveryID, map[string]interface{}{
		"stage":        enum.DeliveryStageDelivered,
		"delivered_at": now,
	})
}
