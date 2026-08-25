package service

import (
	"time"

	"gorm.io/gorm"

	"photography-server/internal/model"
	"photography-server/internal/pkg/errs"
)

type DeliveryItemReq struct {
	URL      string `json:"url" binding:"required"`
	FileType string `json:"file_type"`
	Kind     string `json:"kind"`
	Filename string `json:"filename"`
	Size     int64  `json:"size"`
}

func (s *Service) GetDelivery(op Operator, orderID int64) (*model.Delivery, error) {
	var d model.Delivery
	if err := s.tenant(op).Where("order_id = ?", orderID).First(&d).Error; err != nil {
		return nil, errs.NotFound("交付单不存在")
	}
	return &d, nil
}

func (s *Service) deliveryItems(op Operator, deliveryID int64) ([]model.DeliveryItem, error) {
	var list []model.DeliveryItem
	err := s.tenant(op).Where("delivery_id = ?", deliveryID).Order("id ASC").Find(&list).Error
	return list, err
}

// ensureDelivery 获取或创建订单的交付单
func (s *Service) ensureDelivery(tx *gorm.DB, op Operator, orderID int64) (*model.Delivery, error) {
	var d model.Delivery
	if err := tx.Where("company_id = ? AND order_id = ?", op.CompanyID, orderID).First(&d).Error; err == nil {
		return &d, nil
	}
	var o model.Order
	if err := tx.First(&o, orderID).Error; err != nil {
		return nil, errs.NotFound("订单不存在")
	}
	d = model.Delivery{
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
	if err := tx.Create(&d).Error; err != nil {
		return nil, err
	}
	return &d, nil
}

// UploadSamples 上传样片：待上传样片 -> 客户选片中
func (s *Service) UploadSamples(op Operator, orderID int64, items []DeliveryItemReq) error {
	return s.DB.Transaction(func(tx *gorm.DB) error {
		d, err := s.ensureDelivery(tx, op, orderID)
		if err != nil {
			return err
		}
		stage := d.Stage
		if stage == model.DeliveryStageUploadPending {
			stage = model.DeliveryStageSelecting
		}
		for _, it := range items {
			if err := tx.Create(&model.DeliveryItem{
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
			}).Error; err != nil {
				return err
			}
		}
		var sampleCount int64
		tx.Model(&model.DeliveryItem{}).Where("delivery_id = ? AND kind = ?", d.ID, "sample").Count(&sampleCount)
		return tx.Model(&d).Updates(map[string]interface{}{
			"stage":        stage,
			"sample_count": sampleCount,
			"operator_id":  op.UserID,
			"updated_by":   op.UserID,
		}).Error
	})
}

// SelectPhotos 客户选片：标记已选，进入精修进行中
func (s *Service) SelectPhotos(op Operator, deliveryID int64, itemIDs []int64) error {
	return s.DB.Transaction(func(tx *gorm.DB) error {
		var d model.Delivery
		if err := tx.Where("company_id = ? AND id = ?", op.CompanyID, deliveryID).First(&d).Error; err != nil {
			return errs.NotFound("交付单不存在")
		}
		if d.Stage != model.DeliveryStageSelecting {
			return errs.BadRequest("当前阶段不可选片")
		}
		if len(itemIDs) > 0 {
			if err := tx.Model(&model.DeliveryItem{}).Where("delivery_id = ? AND id IN ?", deliveryID, itemIDs).
				Update("kind", "selected").Error; err != nil {
				return err
			}
		}
		var selected int64
		tx.Model(&model.DeliveryItem{}).Where("delivery_id = ? AND kind = ?", deliveryID, "selected").Count(&selected)
		now := time.Now().Format("2006-01-02 15:04:05")
		return tx.Model(&d).Updates(map[string]interface{}{
			"stage":          model.DeliveryStageRetouching,
			"selected_count": selected,
			"selected_at":    now,
			"operator_id":    op.UserID,
			"updated_by":     op.UserID,
		}).Error
	})
}

// UploadRetouched 上传精修成品：进入待确认交付
func (s *Service) UploadRetouched(op Operator, deliveryID int64, items []DeliveryItemReq) error {
	return s.DB.Transaction(func(tx *gorm.DB) error {
		var d model.Delivery
		if err := tx.Where("company_id = ? AND id = ?", op.CompanyID, deliveryID).First(&d).Error; err != nil {
			return errs.NotFound("交付单不存在")
		}
		for _, it := range items {
			if err := tx.Create(&model.DeliveryItem{
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
			}).Error; err != nil {
				return err
			}
		}
		var retouched int64
		tx.Model(&model.DeliveryItem{}).Where("delivery_id = ? AND kind = ?", deliveryID, "retouched").Count(&retouched)
		return tx.Model(&d).Updates(map[string]interface{}{
			"stage":           model.DeliveryStageDeliverReady,
			"retouched_count": retouched,
			"operator_id":     op.UserID,
			"updated_by":      op.UserID,
		}).Error
	})
}

// ConfirmDelivered 确认交付：交付单完成，订单同步完成
func (s *Service) ConfirmDelivered(op Operator, deliveryID int64) error {
	return s.DB.Transaction(func(tx *gorm.DB) error {
		var d model.Delivery
		if err := tx.Where("company_id = ? AND id = ?", op.CompanyID, deliveryID).First(&d).Error; err != nil {
			return errs.NotFound("交付单不存在")
		}
		if d.Stage != model.DeliveryStageDeliverReady {
			return errs.BadRequest("当前阶段不可确认交付")
		}
		now := time.Now().Format("2006-01-02 15:04:05")
		if err := tx.Model(&d).Updates(map[string]interface{}{
			"stage":        model.DeliveryStageCompleted,
			"delivered_at": now,
			"operator_id":  op.UserID,
			"updated_by":   op.UserID,
		}).Error; err != nil {
			return err
		}
		return s.writeOrderLogTx(tx, d.OrderID, "delivery_done", "", "", "交付完成", op)
	})
}

func (s *Service) ListDeliveryItems(op Operator, deliveryID int64) ([]model.DeliveryItem, error) {
	return s.deliveryItems(op, deliveryID)
}
