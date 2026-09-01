package service

import (
	"errors"
	"time"

	"gorm.io/gorm"

	"photography-server/internal/common"
	"photography-server/internal/model"
	"photography-server/internal/pkg/errs"
	"photography-server/internal/presentation/dto"
)

func (s *Service) GetDelivery(op Operator, orderID int64) (*model.Delivery, error) {
	d, err := s.DeliveryRepo.GetByOrderID(op.CompanyID, orderID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.NotFound(common.ErrDeliveryNotFound)
		}
		return nil, err
	}
	return d, nil
}

// ensureDelivery 获取或创建订单的交付单
func (s *Service) ensureDelivery(tx *gorm.DB, op Operator, orderID int64) (*model.Delivery, error) {
	o, err := s.OrderRepo.GetByID(op.CompanyID, orderID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.NotFound(common.ErrOrderNotFound)
		}
		return nil, err
	}
	d := model.Delivery{
		TenantBase: model.TenantBase{
			Base:      model.Base{CreatedBy: op.UserID, UpdatedBy: op.UserID},
			CompanyID: op.CompanyID,
		},
		Code:         genCode("DV"),
		OrderID:      orderID,
		CustomerID:   o.CustomerID,
		CustomerName: o.CustomerName,
		Stage:        model.DeliveryStageUploadPending,
		OperatorID:   op.UserID,
	}
	return s.DeliveryRepo.GetOrCreateByOrder(tx, op.CompanyID, orderID, &d)
}

// UploadSamples 上传样片：待上传样片 -> 客户选片中
func (s *Service) UploadSamples(op Operator, orderID int64, items []dto.DeliveryItemReq) error {
	return s.DB().Transaction(func(tx *gorm.DB) error {
		d, err := s.ensureDelivery(tx, op, orderID)
		if err != nil {
			return err
		}
		stage := d.Stage
		if stage == model.DeliveryStageUploadPending {
			stage = model.DeliveryStageSelecting
		}
		for _, it := range items {
			if err := s.DeliveryRepo.CreateItem(tx, &model.DeliveryItem{
				TenantBase: model.TenantBase{
					Base:      model.Base{CreatedBy: op.UserID, UpdatedBy: op.UserID},
					CompanyID: op.CompanyID,
				},
				DeliveryID: d.ID,
				OrderID:    orderID,
				URL:        it.URL,
				FileType:   it.FileType,
				Kind:       "sample",
				Filename:   it.Filename,
				Size:       it.Size,
			}); err != nil {
				return err
			}
		}
		sampleCount, _ := s.DeliveryRepo.CountItemsByKind(op.CompanyID, d.ID, "sample")
		return s.DeliveryRepo.Update(op.CompanyID, d.ID, map[string]interface{}{
			"stage":        stage,
			"sample_count": sampleCount,
			"operator_id":  op.UserID,
			"updated_by":   op.UserID,
		})
	})
}

// SelectPhotos 客户选片：标记已选，进入精修进行中
func (s *Service) SelectPhotos(op Operator, deliveryID int64, itemIDs []int64) error {
	return s.DB().Transaction(func(tx *gorm.DB) error {
		d, err := s.DeliveryRepo.GetByID(op.CompanyID, deliveryID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errs.NotFound(common.ErrDeliveryNotFound)
			}
			return err
		}
		if d.Stage != model.DeliveryStageSelecting {
			return errs.BadRequest(common.ErrDeliveryStageInvalid)
		}
		if len(itemIDs) > 0 {
			if err := s.DeliveryRepo.UpdateItemKind(op.CompanyID, deliveryID, itemIDs, "selected"); err != nil {
				return err
			}
		}
		selected, _ := s.DeliveryRepo.CountItemsByKind(op.CompanyID, deliveryID, "selected")
		now := time.Now().Format("2006-01-02 15:04:05")
		return s.DeliveryRepo.Update(op.CompanyID, deliveryID, map[string]interface{}{
			"stage":          model.DeliveryStageRetouching,
			"selected_count": selected,
			"selected_at":    now,
			"operator_id":    op.UserID,
			"updated_by":     op.UserID,
		})
	})
}

// UploadRetouched 上传精修成品：进入待确认交付
func (s *Service) UploadRetouched(op Operator, deliveryID int64, items []dto.DeliveryItemReq) error {
	return s.DB().Transaction(func(tx *gorm.DB) error {
		d, err := s.DeliveryRepo.GetByID(op.CompanyID, deliveryID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errs.NotFound(common.ErrDeliveryNotFound)
			}
			return err
		}
		for _, it := range items {
			if err := s.DeliveryRepo.CreateItem(tx, &model.DeliveryItem{
				TenantBase: model.TenantBase{
					Base:      model.Base{CreatedBy: op.UserID, UpdatedBy: op.UserID},
					CompanyID: op.CompanyID,
				},
				DeliveryID: deliveryID,
				OrderID:    d.OrderID,
				URL:        it.URL,
				FileType:   it.FileType,
				Kind:       "retouched",
				Filename:   it.Filename,
				Size:       it.Size,
			}); err != nil {
				return err
			}
		}
		retouched, _ := s.DeliveryRepo.CountItemsByKind(op.CompanyID, deliveryID, "retouched")
		return s.DeliveryRepo.Update(op.CompanyID, deliveryID, map[string]interface{}{
			"stage":           model.DeliveryStageDeliverReady,
			"retouched_count": retouched,
			"operator_id":     op.UserID,
			"updated_by":      op.UserID,
		})
	})
}

// ConfirmDelivered 确认交付：交付单完成，订单同步完成
func (s *Service) ConfirmDelivered(op Operator, deliveryID int64) error {
	return s.DB().Transaction(func(tx *gorm.DB) error {
		d, err := s.DeliveryRepo.GetByID(op.CompanyID, deliveryID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errs.NotFound(common.ErrDeliveryNotFound)
			}
			return err
		}
		if d.Stage != model.DeliveryStageDeliverReady {
			return errs.BadRequest(common.ErrDeliveryStageInvalid)
		}
		now := time.Now().Format("2006-01-02 15:04:05")
		if err := s.DeliveryRepo.Update(op.CompanyID, deliveryID, map[string]interface{}{
			"stage":        model.DeliveryStageCompleted,
			"delivered_at": now,
			"operator_id":  op.UserID,
			"updated_by":   op.UserID,
		}); err != nil {
			return err
		}
		return s.writeOrderLogTx(tx, d.OrderID, "delivery_done", "", "", "交付完成", op)
	})
}

func (s *Service) ListDeliveryItems(op Operator, deliveryID int64) ([]model.DeliveryItem, error) {
	return s.DeliveryRepo.ListItems(op.CompanyID, deliveryID)
}
