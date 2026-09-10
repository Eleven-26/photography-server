package service

import (
	"context"
	"time"

	"photography-server/internal/domain"
	"photography-server/internal/enum"
	"photography-server/internal/model"
	"photography-server/internal/pkg/errs"
	"photography-server/internal/presentation/dto"
	"photography-server/internal/repository"
)

func (s *Service) CreateDelivery(ctx context.Context, op Operator, orderID int64) (*model.Delivery, error) {
	return s.CreateDeliveryTask(ctx, op, dto.DeliveryCreateReq{OrderID: orderID})
}

// CreateDeliveryTask 新建交付任务（PC 交付工作台 / 员工端共用）。
// 同一订单只允许一张交付单：已存在时直接返回原单，避免重复建单导致选片链路分裂。
func (s *Service) CreateDeliveryTask(ctx context.Context, op Operator, req dto.DeliveryCreateReq) (*model.Delivery, error) {
	if req.OrderID <= 0 {
		return nil, errs.BadRequest("订单ID无效")
	}
	o, err := s.OrderRepo.GetByID(ctx, op.CompanyID, req.OrderID)
	if err != nil {
		return nil, errs.NotFound(errs.ErrOrderNotFound)
	}
	if existed, err := s.DeliveryRepo.GetByOrderID(ctx, op.CompanyID, req.OrderID); err == nil && existed != nil {
		return existed, nil
	}

	stage := enum.DeliveryStage(orDefaultInt64(int64(req.Stage), int64(enum.DeliveryStagePendingSamples)))
	if stage < enum.DeliveryStagePendingSamples || stage > enum.DeliveryStagePendingConfirm {
		return nil, errs.BadRequest("交付阶段参数错误")
	}

	now := time.Now()
	d := model.Delivery{
		TenantBase: model.TenantBase{
			Base:      model.Base{CreatedAt: now, UpdatedAt: now, CreatedBy: op.UserID, UpdatedBy: op.UserID},
			CompanyID: op.CompanyID,
		},
		Code:           domain.GenCode("DV"),
		OrderID:        req.OrderID,
		CustomerID:     o.CustomerID,
		CustomerName:   o.CustomerName,
		Stage:          stage,
		RawCount:       req.RawCount,
		RetouchTarget:  req.RetouchTarget,
		SelectDeadline: strPtr(req.SelectDeadline),
		Remark:         req.Remark,
		OperatorID:     req.OperatorID,
	}
	if err := s.DeliveryRepo.Create(ctx, &d); err != nil {
		return nil, err
	}
	if req.OperatorID > 0 {
		s.NotifyStaff(ctx, op, req.OperatorID, "delivery",
			"新的交付任务",
			"订单 "+o.Code+" 已指派给你，请及时处理",
			"delivery", d.ID)
	}
	return &d, nil
}

// ListDeliveries 交付工作台看板列表。
func (s *Service) ListDeliveries(ctx context.Context, op Operator, stage, page, pageSize int, keyword string) ([]repository.DeliveryListItem, int64, error) {
	return s.DeliveryRepo.List(ctx, op.CompanyID, stage, keyword, page, pageSize)
}

// RemindDeliveryOperator 提醒交付负责人。未指派（operator_id=0）时广播给公司内启用员工。
func (s *Service) RemindDeliveryOperator(ctx context.Context, op Operator, deliveryID int64) error {
	d, err := s.DeliveryRepo.GetByID(ctx, op.CompanyID, deliveryID)
	if err != nil {
		return errs.NotFound("交付单不存在")
	}
	s.NotifyStaff(ctx, op, d.OperatorID, "delivery",
		"交付任务提醒",
		"交付单 "+d.Code+"（客户 "+d.CustomerName+"）请尽快跟进",
		"delivery", d.ID)
	return nil
}

func (s *Service) GetDeliveryByOrder(ctx context.Context, op Operator, orderID int64) (*model.Delivery, error) {
	return s.DeliveryRepo.GetByOrderID(ctx, op.CompanyID, orderID)
}

func (s *Service) UploadSamples(ctx context.Context, op Operator, deliveryID int64, items []dto.DeliveryItemReq) error {
	d, err := s.DeliveryRepo.GetByID(ctx, op.CompanyID, deliveryID)
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
		if err := s.DeliveryRepo.CreateItem(ctx, &di); err != nil {
			return err
		}
	}

	if err := s.DeliveryRepo.Update(ctx, op.CompanyID, deliveryID, map[string]interface{}{
		"stage":        enum.DeliveryStageSelecting,
		"sample_count": len(items),
	}); err != nil {
		return err
	}
	// 选片是有截止时间的客户待办，样片就绪必须通知到客户本人
	s.NotifyClient(ctx, op, d.CustomerID, "order", "样片已上传，可开始选片",
		"交付单 "+d.Code+" 的样片已上传，请在选片截止前完成选片", "delivery", d.ID)
	return nil
}

func (s *Service) SelectPhotos(ctx context.Context, op Operator, deliveryID int64, req dto.DeliverySelectReq) error {
	d, err := s.DeliveryRepo.GetByID(ctx, op.CompanyID, deliveryID)
	if err != nil {
		return errs.NotFound(errs.ErrDeliveryNotFound)
	}
	if d.Stage != enum.DeliveryStageSelecting {
		return errs.BadRequest(errs.ErrDeliveryStageInvalid)
	}

	for _, itemID := range req.ItemIDs {
		s.DeliveryRepo.UpdateItemKind(ctx, op.CompanyID, itemID, "selected")
	}

	now := time.Now().Format("2006-01-02 15:04:05")
	return s.DeliveryRepo.Update(ctx, op.CompanyID, deliveryID, map[string]interface{}{
		"stage":          enum.DeliveryStageRetouching,
		"selected_count": len(req.ItemIDs),
		"selected_at":    now,
	})
}

func (s *Service) UploadRetouched(ctx context.Context, op Operator, deliveryID int64, items []dto.DeliveryItemReq) error {
	d, err := s.DeliveryRepo.GetByID(ctx, op.CompanyID, deliveryID)
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
		if err := s.DeliveryRepo.CreateItem(ctx, &di); err != nil {
			return err
		}
	}

	return s.DeliveryRepo.Update(ctx, op.CompanyID, deliveryID, map[string]interface{}{
		"stage":           enum.DeliveryStagePendingConfirm,
		"retouched_count": len(items),
	})
}

func (s *Service) ConfirmDelivered(ctx context.Context, op Operator, deliveryID int64) error {
	d, err := s.DeliveryRepo.GetByID(ctx, op.CompanyID, deliveryID)
	if err != nil {
		return errs.NotFound(errs.ErrDeliveryNotFound)
	}
	if d.Stage != enum.DeliveryStagePendingConfirm {
		return errs.BadRequest(errs.ErrDeliveryStageInvalid)
	}

	now := time.Now().Format("2006-01-02 15:04:05")
	if err := s.DeliveryRepo.Update(ctx, op.CompanyID, deliveryID, map[string]interface{}{
		"stage":        enum.DeliveryStageDelivered,
		"delivered_at": now,
	}); err != nil {
		return err
	}
	s.NotifyClient(ctx, op, d.CustomerID, "order", "成片已交付",
		"交付单 "+d.Code+" 的成片已交付，请及时下载保存", "delivery", d.ID)
	return nil
}

// ---------------------------------------------------------------------
// 报告 H11：员工端「反馈整理」——客户在精修阶段提交的修改意见
// ---------------------------------------------------------------------

// ListFeedbackItems 客户反馈列表（status 见 enum.Feedback*，0 为全部）
func (s *Service) ListFeedbackItems(ctx context.Context, op Operator, status, page, pageSize int) ([]repository.FeedbackListItem, int64, error) {
	return s.DeliveryRepo.ListFeedbackItems(ctx, op.CompanyID, status, page, pageSize)
}

// HandleFeedbackItem 标记反馈已处理并记录处理备注（如「已按要求重修」）。
// 无反馈的文件不可操作；重复提交按幂等返回成功（避免双击报错）。
func (s *Service) HandleFeedbackItem(ctx context.Context, op Operator, itemID int64, remark string) error {
	var item model.DeliveryItem
	if err := s.DeliveryRepo.FirstItem(ctx, op.CompanyID, itemID, &item); err != nil {
		return errs.NotFound("交付文件不存在")
	}
	if item.FeedbackStatus == enum.FeedbackNone {
		return errs.BadRequest("该文件没有客户反馈")
	}
	if item.FeedbackStatus == enum.FeedbackHandled {
		return nil
	}
	return s.DeliveryRepo.UpdateItem(ctx, op.CompanyID, itemID, map[string]interface{}{
		"feedback_status": enum.FeedbackHandled,
		"handled_at":      time.Now().Format("2006-01-02 15:04:05"),
		"handle_remark":   remark,
		"updated_by":      op.UserID,
	})
}
