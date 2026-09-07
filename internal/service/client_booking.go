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

// client_booking 客户端（H5/小程序）预约链路：
// 提交预约（待确认）→ 工作室确认 → 待定金；提交/取消均为客户本人操作，带归属校验。

// clientOperator 客户操作映射为订单日志的操作人
func clientOperator(cu *ClientUser) Operator {
	return Operator{UserID: cu.CustomerID, Username: cu.Name, CompanyID: cu.CompanyID}
}

// clientOrderOwned 校验订单归属当前登录客户
func clientOrderOwned(o *model.Order, cu *ClientUser) error {
	if o.CustomerID != cu.CustomerID {
		return errs.Forbidden("无权操作该订单")
	}
	return nil
}

// ClientSubmitBooking 客户提交预约单（来源=客户预约，状态=待确认）。
// 事务内：建线索 → 建订单（快照）→ 锁档期 → 写线索消息 → 订单日志。
func (s *Service) ClientSubmitBooking(ctx context.Context, cu *ClientUser, req dto.ClientBookingReq) (*model.Order, error) {
	if req.PackageID <= 0 {
		return nil, errs.BadRequest("请选择套餐")
	}
	if req.ShootDate == "" || req.ShootTime == "" {
		return nil, errs.BadRequest("请选择拍摄日期与时段")
	}
	pkg, err := s.PackageRepo.GetByID(ctx, cu.CompanyID, req.PackageID)
	if err != nil || pkg.Status != enum.PackageStatusActive {
		return nil, errs.NotFound(errs.ErrPackageNotFound)
	}

	// 工作室暂停接单时拒绝
	st, err := s.StudioSettingRepo.GetOrCreate(ctx, cu.CompanyID)
	if err != nil {
		return nil, err
	}
	if st.AcceptNew == 0 {
		return nil, errs.BadRequest("工作室已暂停接单，请稍后再试")
	}

	// 时段占用检查（同日同时段已有锁定档期则拒绝）
	blocks, err := s.CalendarRepo.List(ctx, cu.CompanyID, req.ShootDate, req.ShootDate, 0)
	if err != nil {
		return nil, err
	}
	for _, b := range blocks {
		if b.Status != enum.BlockStatusCancelled && (b.TimeRange == req.ShootTime || b.TimeRange == "") {
			return nil, errs.BadRequest("该时段已被预约，请选择其他时段")
		}
	}

	now := time.Now()
	// 线索：客户预约自动建档，负责人待工作室分配（OwnerID=0）
	lead := model.Lead{
		TenantBase:  model.TenantBase{Base: model.Base{CreatedAt: now, UpdatedAt: now}, CompanyID: cu.CompanyID},
		Code:        domain.GenCode("LD"),
		CustomerID:  cu.CustomerID,
		Name:        cu.Name,
		Mobile:      cu.Mobile,
		Source:      "预约主页",
		ProjectType: pkg.Category,
		ShootDate:   strPtr(req.ShootDate),
		Remark:      req.Remark,
		Status:      enum.LeadStatusPending,
	}

	o := model.Order{
		TenantBase:     model.TenantBase{Base: model.Base{CreatedAt: now, UpdatedAt: now}, CompanyID: cu.CompanyID},
		Code:           domain.GenCode("SL"),
		CustomerID:     cu.CustomerID,
		CustomerName:   cu.Name,
		CustomerMobile: cu.Mobile,
		PackageID:      pkg.ID,
		PackageName:    pkg.Name,
		PackageVersion: pkg.Version,
		BasePrice:      pkg.BasePrice,
		DepositAmt:     domain.Round2(pkg.BasePrice * pkg.DepositRate / 100),
		FinalAmt:       domain.Round2(pkg.BasePrice - pkg.BasePrice*pkg.DepositRate/100),
		TotalAmt:       pkg.BasePrice,
		ShootDate:      strPtr(req.ShootDate),
		ShootTime:      req.ShootTime,
		ShootAddress:   req.ShootAddress,
		PeopleCount:    req.PeopleCount,
		ShootStyle:     req.ShootStyle,
		SourceType:     2, // 客户预约
		Status:         enum.OrderStatusPendingConfirm,
		PaymentStatus:  enum.PaymentStatusPending,
		Remark:         req.Remark,
	}

	err = repository.Tx(func(tx *gorm.DB) error {
		if err := s.LeadRepo.WithTx(tx).Create(ctx, &lead); err != nil {
			return err
		}
		o.LeadID = lead.ID
		if err := s.OrderRepo.WithTx(tx).Create(ctx, &o); err != nil {
			return err
		}
		// 回写线索/报价关联（客户预约走套餐直订，无报价单）
		if err := s.LeadRepo.WithTx(tx).Update(ctx, cu.CompanyID, lead.ID, map[string]interface{}{"status": enum.LeadStatusQuoting}); err != nil {
			return err
		}
		block := model.CalendarBlock{
			TenantBase:   model.TenantBase{Base: model.Base{CreatedAt: now, UpdatedAt: now}, CompanyID: cu.CompanyID},
			OrderID:      o.ID,
			CustomerID:   cu.CustomerID,
			CustomerName: cu.Name,
			Date:         req.ShootDate,
			TimeRange:    req.ShootTime,
			ProjectType:  pkg.Category,
			Status:       enum.BlockStatusLocked,
		}
		if err := s.CalendarRepo.WithTx(tx).Create(ctx, &block); err != nil {
			return err
		}
		msg := model.LeadMessage{
			TenantBase: model.TenantBase{Base: model.Base{CreatedAt: now, UpdatedAt: now}, CompanyID: cu.CompanyID},
			LeadID:     lead.ID,
			CustomerID: cu.CustomerID,
			Direction:  1, // 客户发来
			Channel:    "h5",
			Content:    "提交预约：" + pkg.Name + "，" + req.ShootDate + " " + req.ShootTime,
			MsgType:    1,
		}
		if err := s.LeadExtraRepo.WithTx(tx).CreateMessage(ctx, &msg); err != nil {
			return err
		}
		return s.writeOrderLogTx(ctx, tx, o.ID, "client_booking", 0, enum.OrderStatusPendingConfirm, "客户提交预约", clientOperator(cu))
	})
	if err != nil {
		return nil, err
	}
	return &o, nil
}

// ClientConfirmBooking 客户确认预约单：待确认 → 待定金
func (s *Service) ClientConfirmBooking(ctx context.Context, cu *ClientUser, orderID int64) error {
	o, err := s.OrderRepo.GetByID(ctx, cu.CompanyID, orderID)
	if err != nil {
		return errs.NotFound(errs.ErrOrderNotFound)
	}
	if err := clientOrderOwned(o, cu); err != nil {
		return err
	}
	if !domain.OrderCanTransit(o.Status, enum.OrderStatusPendingDeposit) {
		return errs.BadRequest(errs.ErrOrderStatusInvalid)
	}
	if err := s.OrderRepo.Update(ctx, cu.CompanyID, orderID, map[string]interface{}{
		"status": enum.OrderStatusPendingDeposit,
	}); err != nil {
		return err
	}
	return s.writeOrderLog(ctx, orderID, "client_confirm", o.Status, enum.OrderStatusPendingDeposit, "客户确认预约单", clientOperator(cu))
}

// ClientCancelBooking 客户取消预约单：待确认 → 已取消（同步取消档期锁）
func (s *Service) ClientCancelBooking(ctx context.Context, cu *ClientUser, orderID int64, reason string) error {
	o, err := s.OrderRepo.GetByID(ctx, cu.CompanyID, orderID)
	if err != nil {
		return errs.NotFound(errs.ErrOrderNotFound)
	}
	if err := clientOrderOwned(o, cu); err != nil {
		return err
	}
	if o.Status != enum.OrderStatusPendingConfirm {
		return errs.BadRequest("当前状态不可取消，如需取消请联系工作室")
	}
	err = repository.Tx(func(tx *gorm.DB) error {
		if err := s.OrderRepo.WithTx(tx).Update(ctx, cu.CompanyID, orderID, map[string]interface{}{
			"status":        enum.OrderStatusCancelled,
			"cancel_reason": reason,
		}); err != nil {
			return err
		}
		if err := s.OrderRepo.WithTx(tx).UpdateCalendarBlockStatus(ctx, cu.CompanyID, orderID, enum.BlockStatusCancelled); err != nil {
			return err
		}
		return s.writeOrderLogTx(ctx, tx, orderID, "client_cancel", o.Status, enum.OrderStatusCancelled, "客户取消预约: "+reason, clientOperator(cu))
	})
	return err
}

// ClientOrders 我的订单列表（仅本人）
func (s *Service) ClientOrders(ctx context.Context, cu *ClientUser, page, pageSize int, status string) ([]model.Order, int64, error) {
	return s.OrderRepo.List(ctx, cu.CompanyID, page, pageSize, status, cu.CustomerID)
}

// ClientOrderDetail 我的订单详情（含收款/退款/交付/改期/加项/评价）
func (s *Service) ClientOrderDetail(ctx context.Context, cu *ClientUser, orderID int64) (*dto.ClientOrderDetail, error) {
	o, err := s.OrderRepo.GetByID(ctx, cu.CompanyID, orderID)
	if err != nil {
		return nil, errs.NotFound(errs.ErrOrderNotFound)
	}
	if err := clientOrderOwned(o, cu); err != nil {
		return nil, err
	}
	payments, _ := s.OrderRepo.ListPayments(ctx, cu.CompanyID, orderID)
	refunds, _ := s.OrderRepo.ListRefunds(ctx, cu.CompanyID, orderID)
	logs, _ := s.OrderRepo.ListLogs(ctx, cu.CompanyID, orderID)
	delivery, _ := s.DeliveryRepo.GetByOrderID(ctx, cu.CompanyID, orderID)
	reschedules, _ := s.RescheduleRepo.ListByOrder(ctx, cu.CompanyID, orderID)
	addons, _ := s.AddonRepo.ListByOrder(ctx, cu.CompanyID, orderID)
	review, _ := s.ReviewRepo.GetByOrderID(ctx, cu.CompanyID, orderID)

	return &dto.ClientOrderDetail{
		Order:       o,
		Payments:    payments,
		Refunds:     refunds,
		Logs:        logs,
		Delivery:    delivery,
		Reschedules: reschedules,
		Addons:      addons,
		Review:      review,
	}, nil
}
