package service

import (
	"context"
	"time"

	"gorm.io/gorm"

	"photography-server/internal/domain"
	"photography-server/internal/enum"
	"photography-server/internal/model"
	"photography-server/internal/pkg/errs"
	"photography-server/internal/presentation/dto"
	"photography-server/internal/repository"
)

// client_order 客户端（H5/小程序）订单后续操作：
// 改期申请、退款申请、评价、选片、确认加片、确认成片。全部带客户归属校验。

// ClientRescheduleApply 客户发起改期申请（同一订单同时最多一张待确认改期单）。
// 费用按工作室改期政策计算：>=FreeHours 免费、区间收调度费、<MinHours 拒绝。
func (s *Service) ClientRescheduleApply(ctx context.Context, cu *ClientUser, orderID int64, req dto.ClientRescheduleReq) (*model.OrderReschedule, error) {
	o, err := s.OrderRepo.GetByID(ctx, cu.CompanyID, orderID)
	if err != nil {
		return nil, errs.NotFound(errs.ErrOrderNotFound)
	}
	if err := clientOrderOwned(o, cu); err != nil {
		return nil, err
	}
	switch o.Status {
	case enum.OrderStatusPendingConfirm, enum.OrderStatusPendingDeposit, enum.OrderStatusPendingShoot:
		// 可改期状态
	default:
		return nil, errs.BadRequest("当前订单状态不可改期")
	}
	if req.NewDate == "" || req.NewTime == "" {
		return nil, errs.BadRequest("请选择新的拍摄日期与时段")
	}
	if _, err := s.RescheduleRepo.GetPendingByOrder(ctx, cu.CompanyID, orderID); err == nil {
		return nil, errs.BadRequest("已有待确认的改期申请，请耐心等待")
	}

	// 距原拍摄开始的小时数 → 费用档位
	origDate := ""
	if o.ShootDate != nil {
		origDate = *o.ShootDate
	}
	origStart, _ := time.ParseInLocation("2006-01-02 15:04", origDate+" 00:00", time.Local)
	if hourPart(o.ShootTime) != "" {
		if t, err := time.ParseInLocation("2006-01-02 15:04", origDate+" "+hourPart(o.ShootTime), time.Local); err == nil {
			origStart = t
		}
	}
	feeType, feeAmt, _ := domain.RescheduleFee(time.Until(origStart), o.TotalAmt, s.clientReschedulePolicy(ctx, cu.CompanyID))
	if feeType == enum.RescheduleFeeForbidden {
		return nil, errs.BadRequest("距拍摄不足24小时，不可改期，请联系工作室")
	}

	now := time.Now()
	rs := model.OrderReschedule{
		TenantBase:   model.TenantBase{Base: model.Base{CreatedAt: now, UpdatedAt: now}, CompanyID: cu.CompanyID},
		Code:         domain.GenCode("RS"),
		OrderID:      orderID,
		CustomerID:   cu.CustomerID,
		OriginalDate: origDate,
		OriginalTime: o.ShootTime,
		NewDate:      req.NewDate,
		NewTime:      req.NewTime,
		FeeType:      feeType,
		FeeAmount:    feeAmt,
		ReasonLabel:  req.ReasonLabel,
		Reason:       req.Reason,
		Status:       enum.RescheduleStatusPending,
		ApplySource:  2, // 客户申请
	}
	if err := s.RescheduleRepo.Create(ctx, &rs); err != nil {
		return nil, err
	}
	_ = s.writeOrderLog(ctx, orderID, "client_reschedule", o.Status, o.Status,
		"客户申请改期至 "+req.NewDate+" "+req.NewTime, clientOperator(cu))
	return &rs, nil
}

// hourPart 从 "09:00-11:00" 中取开始小时部分
func hourPart(timeRange string) string {
	for i := 0; i < len(timeRange); i++ {
		if timeRange[i] == '-' {
			return timeRange[:i]
		}
	}
	return timeRange
}

// ClientRescheduleCancel 客户撤回待确认的改期申请
func (s *Service) ClientRescheduleCancel(ctx context.Context, cu *ClientUser, rescheduleID int64) error {
	rs, err := s.RescheduleRepo.GetByID(ctx, cu.CompanyID, rescheduleID)
	if err != nil {
		return errs.NotFound("改期单不存在")
	}
	if rs.CustomerID != cu.CustomerID {
		return errs.Forbidden("无权操作该改期单")
	}
	if rs.Status != enum.RescheduleStatusPending {
		return errs.BadRequest("该改期单已处理，不可撤回")
	}
	return s.RescheduleRepo.Update(ctx, cu.CompanyID, rescheduleID, map[string]interface{}{
		"status": enum.RescheduleStatusCancelled,
	})
}

// ClientRefundApply 客户申请退款（apply_source=2），金额按退款规则试算
func (s *Service) ClientRefundApply(ctx context.Context, cu *ClientUser, orderID int64, req dto.ClientRefundReq) (*model.OrderRefund, error) {
	o, err := s.OrderRepo.GetByID(ctx, cu.CompanyID, orderID)
	if err != nil {
		return nil, errs.NotFound(errs.ErrOrderNotFound)
	}
	if err := clientOrderOwned(o, cu); err != nil {
		return nil, err
	}
	if o.Status == enum.OrderStatusCancelled {
		return nil, errs.BadRequest(errs.ErrRefundCancelled)
	}
	if o.PaidAmt <= 0 {
		return nil, errs.BadRequest(errs.ErrRefundNoTime)
	}

	// 按距拍摄时间试算可退金额
	shootDate := ""
	if o.ShootDate != nil {
		shootDate = *o.ShootDate
	}
	shootTime, _ := time.Parse("2006-01-02", shootDate)
	hours := time.Until(shootTime).Hours()
	preview := domain.CalcRefundPreview(time.Duration(hours) * time.Hour)
	refundAmt := domain.Round2(o.PaidAmt * preview.Ratio)
	if refundAmt <= 0 {
		return nil, errs.BadRequest(errs.ErrRefundNoTime)
	}

	now := time.Now()
	rf := model.OrderRefund{
		TenantBase:  model.TenantBase{Base: model.Base{CreatedAt: now, UpdatedAt: now}, CompanyID: cu.CompanyID},
		OrderID:     orderID,
		Code:        domain.GenCode("RF"),
		CustomerID:  cu.CustomerID,
		Amount:      refundAmt,
		Reason:      req.Reason,
		ReasonLabel: req.ReasonLabel,
		ApplySource: 2, // 客户H5/小程序
		RefundRule:  preview.Rule,
		Status:      enum.RefundStatusApplying,
		ApplyBy:     cu.CustomerID,
		ApplyName:   cu.Name,
	}
	if err := s.OrderRepo.CreateRefund(ctx, &rf); err != nil {
		return nil, err
	}
	_ = s.writeOrderLog(ctx, orderID, "client_refund", o.Status, o.Status, "客户申请退款", clientOperator(cu))
	return &rf, nil
}

// ClientReviewCreate 订单评价（已完成订单、每单一评）
func (s *Service) ClientReviewCreate(ctx context.Context, cu *ClientUser, orderID int64, req dto.ClientReviewReq) (*model.OrderReview, error) {
	o, err := s.OrderRepo.GetByID(ctx, cu.CompanyID, orderID)
	if err != nil {
		return nil, errs.NotFound(errs.ErrOrderNotFound)
	}
	if err := clientOrderOwned(o, cu); err != nil {
		return nil, err
	}
	if o.Status != enum.OrderStatusCompleted {
		return nil, errs.BadRequest("订单完成后方可评价")
	}
	if req.Rating < 1 || req.Rating > 5 {
		return nil, errs.BadRequest("评分需在 1-5 之间")
	}
	if _, err := s.ReviewRepo.GetByOrderID(ctx, cu.CompanyID, orderID); err == nil {
		return nil, errs.BadRequest("该订单已评价过")
	}
	now := time.Now()
	name := cu.Name
	if req.IsAnonymous == 1 {
		name = "匿名用户"
	}
	rv := model.OrderReview{
		TenantBase:   model.TenantBase{Base: model.Base{CreatedAt: now, UpdatedAt: now}, CompanyID: cu.CompanyID},
		OrderID:      orderID,
		CustomerID:   cu.CustomerID,
		CustomerName: name,
		Rating:       req.Rating,
		Content:      req.Content,
		Images:       req.Images,
		IsAnonymous:  req.IsAnonymous,
	}
	if err := s.ReviewRepo.Create(ctx, &rv); err != nil {
		return nil, err
	}
	return &rv, nil
}

// ClientDeliveryItems 客户查看交付明细（选片页/成片页）
func (s *Service) ClientDeliveryItems(ctx context.Context, cu *ClientUser, deliveryID int64) (*model.Delivery, []model.DeliveryItem, error) {
	d, err := s.DeliveryRepo.GetByID(ctx, cu.CompanyID, deliveryID)
	if err != nil {
		return nil, nil, errs.NotFound(errs.ErrDeliveryNotFound)
	}
	o, err := s.OrderRepo.GetByID(ctx, cu.CompanyID, d.OrderID)
	if err != nil {
		return nil, nil, errs.NotFound(errs.ErrOrderNotFound)
	}
	if err := clientOrderOwned(o, cu); err != nil {
		return nil, nil, err
	}
	items, err := s.DeliveryRepo.ListItems(ctx, cu.CompanyID, deliveryID)
	if err != nil {
		return nil, nil, err
	}
	return d, items, nil
}

// ClientSelectPhotos 客户选片：勾选样片；超出套餐精修张数部分计加片费。
// 规则：选片截止前可反复调整；截止后锁定（服务端兜底，前端倒计时）。
func (s *Service) ClientSelectPhotos(ctx context.Context, cu *ClientUser, deliveryID int64, itemIDs []int64) error {
	d, err := s.DeliveryRepo.GetByID(ctx, cu.CompanyID, deliveryID)
	if err != nil {
		return errs.NotFound(errs.ErrDeliveryNotFound)
	}
	o, err := s.OrderRepo.GetByID(ctx, cu.CompanyID, d.OrderID)
	if err != nil {
		return errs.NotFound(errs.ErrOrderNotFound)
	}
	if err := clientOrderOwned(o, cu); err != nil {
		return err
	}
	if d.Stage != enum.DeliveryStageSelecting {
		return errs.BadRequest(errs.ErrDeliveryStageInvalid)
	}
	if d.SelectDeadline != nil && *d.SelectDeadline != "" {
		if deadline, err := time.ParseInLocation("2006-01-02 15:04:05", *d.SelectDeadline, time.Local); err == nil {
			if time.Now().After(deadline) {
				return errs.BadRequest("选片已截止，如需调整请联系工作室")
			}
		}
	}

	items, err := s.DeliveryRepo.ListItems(ctx, cu.CompanyID, deliveryID)
	if err != nil {
		return err
	}
	selSet := map[int64]bool{}
	for _, id := range itemIDs {
		selSet[id] = true
	}
	selected := 0
	for _, it := range items {
		want := 0
		if selSet[it.ID] {
			want = 1
			selected++
		}
		if it.IsSelected != want {
			if err := s.DeliveryRepo.UpdateItem(ctx, cu.CompanyID, it.ID, map[string]interface{}{"is_selected": want}); err != nil {
				return err
			}
		}
	}

	// 加片费：超出套餐精修张数 × 加片单价
	pkg, _ := s.PackageRepo.GetByID(ctx, cu.CompanyID, o.PackageID)
	included := 0
	unitPrice := 0.0
	if pkg != nil {
		included = pkg.PhotosIncluded
		unitPrice = pkg.AddonUnitPrice
	}
	extraCount, extraFee := domain.ExtraRetouchFee(selected, included, unitPrice)

	now := time.Now().Format("2006-01-02 15:04:05")
	return s.DeliveryRepo.Update(ctx, cu.CompanyID, deliveryID, map[string]interface{}{
		"selected_count":       selected,
		"extra_selected_count": extraCount,
		"extra_fee":            extraFee,
		"selected_at":          now,
	})
}

// ClientConfirmExtra 客户确认加片费用：加片金额计入尾款
func (s *Service) ClientConfirmExtra(ctx context.Context, cu *ClientUser, deliveryID int64) error {
	d, err := s.DeliveryRepo.GetByID(ctx, cu.CompanyID, deliveryID)
	if err != nil {
		return errs.NotFound(errs.ErrDeliveryNotFound)
	}
	o, err := s.OrderRepo.GetByID(ctx, cu.CompanyID, d.OrderID)
	if err != nil {
		return errs.NotFound(errs.ErrOrderNotFound)
	}
	if err := clientOrderOwned(o, cu); err != nil {
		return err
	}
	if d.ExtraConfirmed == 1 {
		return nil
	}
	return repository.Tx(func(tx *gorm.DB) error {
		if err := s.DeliveryRepo.WithTx(tx).Update(ctx, cu.CompanyID, deliveryID, map[string]interface{}{
			"extra_confirmed": 1,
		}); err != nil {
			return err
		}
		// 加片费进订单尾款（addon_amount 累加，final_amt 同步）
		if err := s.OrderRepo.WithTx(tx).Update(ctx, cu.CompanyID, o.ID, map[string]interface{}{
			"addon_amount": gorm.Expr("addon_amount + ?", d.ExtraFee),
			"final_amt":    gorm.Expr("final_amt + ?", d.ExtraFee),
			"total_amt":    gorm.Expr("total_amt + ?", d.ExtraFee),
		}); err != nil {
			return err
		}
		return s.writeOrderLogTx(ctx, tx, o.ID, "client_confirm_extra", o.Status, o.Status,
			"客户确认加片，加收金额已计入尾款", clientOperator(cu))
	})
}

// ClientConfirmDelivery 客户确认成片：交付单 → 已交付，订单 → 已完成
func (s *Service) ClientConfirmDelivery(ctx context.Context, cu *ClientUser, deliveryID int64) error {
	d, err := s.DeliveryRepo.GetByID(ctx, cu.CompanyID, deliveryID)
	if err != nil {
		return errs.NotFound(errs.ErrDeliveryNotFound)
	}
	o, err := s.OrderRepo.GetByID(ctx, cu.CompanyID, d.OrderID)
	if err != nil {
		return errs.NotFound(errs.ErrOrderNotFound)
	}
	if err := clientOrderOwned(o, cu); err != nil {
		return err
	}
	if d.Stage != enum.DeliveryStagePendingConfirm {
		return errs.BadRequest("当前阶段不可确认成片")
	}
	now := time.Now()
	nowStr := now.Format("2006-01-02 15:04:05")
	return repository.Tx(func(tx *gorm.DB) error {
		if err := s.DeliveryRepo.WithTx(tx).Update(ctx, cu.CompanyID, deliveryID, map[string]interface{}{
			"stage":                 enum.DeliveryStageDelivered,
			"customer_confirmed_at": nowStr,
			"delivered_at":          nowStr,
		}); err != nil {
			return err
		}
		if !domain.OrderCanTransit(o.Status, enum.OrderStatusCompleted) {
			return errs.BadRequest(errs.ErrOrderStatusInvalid)
		}
		if err := s.OrderRepo.WithTx(tx).Update(ctx, cu.CompanyID, o.ID, map[string]interface{}{
			"status":      enum.OrderStatusCompleted,
			"finished_at": nowStr,
		}); err != nil {
			return err
		}
		return s.writeOrderLogTx(ctx, tx, o.ID, "client_confirm_delivery", o.Status, enum.OrderStatusCompleted,
			"客户确认成片，订单完成", clientOperator(cu))
	})
}

// ClientFeedbackSubmit 客户提交精修反馈（对某张精修成品提出修改意见）
func (s *Service) ClientFeedbackSubmit(ctx context.Context, cu *ClientUser, itemID int64, req dto.ClientFeedbackReq) error {
	var item model.DeliveryItem
	if err := s.DeliveryRepo.FirstItem(ctx, cu.CompanyID, itemID, &item); err != nil {
		return errs.NotFound("交付文件不存在")
	}
	d, err := s.DeliveryRepo.GetByID(ctx, cu.CompanyID, item.DeliveryID)
	if err != nil {
		return errs.NotFound(errs.ErrDeliveryNotFound)
	}
	o, err := s.OrderRepo.GetByID(ctx, cu.CompanyID, d.OrderID)
	if err != nil {
		return errs.NotFound(errs.ErrOrderNotFound)
	}
	if err := clientOrderOwned(o, cu); err != nil {
		return err
	}
	if req.Content == "" && req.Types == "" {
		return errs.BadRequest("请填写修改意见")
	}
	return s.DeliveryRepo.UpdateItem(ctx, cu.CompanyID, itemID, map[string]interface{}{
		"feedback_content":  req.Content,
		"feedback_types":    req.Types,
		"feedback_priority": orDefault(req.Priority, "normal"),
		"feedback_status":   1,
	})
}
