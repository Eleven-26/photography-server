package service

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"photography-server/internal/domain"
	"photography-server/internal/enum"
	"photography-server/internal/model"
	"photography-server/internal/pkg/errs"
	"photography-server/internal/pkg/logger"
	"photography-server/internal/presentation/dto"
)

// 管理端改期。
//
// 为什么不让 PC 直接改订单的 shoot_date/shoot_time：改期必须同步重排档期锁
// （取消旧锁 + 建新锁），而 order/update 只写订单字段、不动日历，会造成
// "订单已改期但档期还锁在旧日期"。改期统一走改期单链路（与客户申请/员工审批同一套），
// 保证审计痕迹与档期一致性。

// ListOrderReschedules 订单改期单列表
func (s *Service) ListOrderReschedules(ctx context.Context, op Operator, orderID int64) ([]model.OrderReschedule, error) {
	if _, err := s.OrderRepo.GetByID(ctx, op.CompanyID, orderID); err != nil {
		return nil, errs.NotFound(errs.ErrOrderNotFound)
	}
	return s.RescheduleRepo.ListByOrder(ctx, op.CompanyID, orderID)
}

// ApplyOrderReschedule 管理端发起改期（apply_source=1）。
// 费用档位、可改期状态、重复申请校验与客户申请链路保持一致，避免两套规则打架。
func (s *Service) ApplyOrderReschedule(ctx context.Context, op Operator, orderID int64, req dto.RescheduleApplyReq) (*model.OrderReschedule, error) {
	if req.NewDate == "" || req.NewTime == "" {
		return nil, errs.BadRequest("请选择新的拍摄日期与时段")
	}
	o, err := s.OrderRepo.GetByID(ctx, op.CompanyID, orderID)
	if err != nil {
		return nil, errs.NotFound(errs.ErrOrderNotFound)
	}
	switch o.Status {
	case enum.OrderStatusPendingConfirm, enum.OrderStatusPendingDeposit, enum.OrderStatusPendingShoot:
		// 可改期状态
	default:
		return nil, errs.BadRequest("当前订单状态不可改期")
	}
	// 仅"确实没有待确认改期单"才放行；查询出错直接返回，避免约束在故障期失效
	if _, err := s.RescheduleRepo.GetPendingByOrder(ctx, op.CompanyID, orderID); err == nil {
		return nil, errs.BadRequest("已有待确认的改期申请，请先处理")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	origDate := ""
	if o.ShootDate != nil {
		origDate = *o.ShootDate
	}
	origStart, _ := time.ParseInLocation("2006-01-02 15:04", origDate+" 00:00", time.Local)
	if hp := hourPart(o.ShootTime); hp != "" {
		if t, err := time.ParseInLocation("2006-01-02 15:04", origDate+" "+hp, time.Local); err == nil {
			origStart = t
		}
	}
	feeType, feeAmt, _ := domain.RescheduleFee(time.Until(origStart), o.TotalAmt, s.clientReschedulePolicy(ctx, op.CompanyID))
	if feeType == enum.RescheduleFeeForbidden {
		return nil, errs.BadRequest("距拍摄不足 24 小时，不可改期，请与客户协商")
	}

	now := time.Now()
	rs := model.OrderReschedule{
		TenantBase: model.TenantBase{
			Base:      model.Base{CreatedBy: op.UserID, UpdatedBy: op.UserID, CreatedAt: now, UpdatedAt: now},
			CompanyID: op.CompanyID,
		},
		Code:         domain.GenCode("RS"),
		OrderID:      orderID,
		CustomerID:   o.CustomerID,
		OriginalDate: origDate,
		OriginalTime: o.ShootTime,
		NewDate:      req.NewDate,
		NewTime:      req.NewTime,
		FeeType:      feeType,
		FeeAmount:    feeAmt,
		ReasonLabel:  req.ReasonLabel,
		Reason:       req.Reason,
		Status:       enum.RescheduleStatusPending,
		ApplySource:  1, // 管理端发起
	}
	if err := s.RescheduleRepo.Create(ctx, &rs); err != nil {
		return nil, err
	}
	if err := s.writeOrderLog(ctx, orderID, "admin_reschedule", o.Status, o.Status,
		"管理端发起改期至 "+req.NewDate+" "+req.NewTime, op); err != nil {
		logger.Warnf("ApplyOrderReschedule: writeOrderLog failed, orderID=%d, err=%v", orderID, err)
	}
	return &rs, nil
}

// ListDeliveryItemsByOrder 交付文件明细（管理端）。
// 入参与 /delivery/detail/:id 一致，均为 **order_id**（按订单反查交付单），
// 避免同一组路由下两种 ID 语义混用。
// 尚未创建交付单时返回空列表（而非 404），前端「文件管理」tab 直接展示空态。
func (s *Service) ListDeliveryItemsByOrder(ctx context.Context, op Operator, orderID int64) ([]model.DeliveryItem, error) {
	d, err := s.DeliveryRepo.GetByOrderID(ctx, op.CompanyID, orderID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return []model.DeliveryItem{}, nil
		}
		return nil, err
	}
	return s.DeliveryRepo.ListItems(ctx, op.CompanyID, d.ID)
}
