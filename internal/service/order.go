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
	"photography-server/internal/repository"
)

func (s *Service) ListOrders(ctx context.Context, op Operator, page, pageSize int, status string, customerID int64) ([]model.Order, int64, error) {
	return s.OrderRepo.List(ctx, op.CompanyID, page, pageSize, status, customerID)
}

func (s *Service) GetOrderDetail(ctx context.Context, op Operator, id int64) (*dto.OrderDetail, error) {
	o, err := s.OrderRepo.GetByID(ctx, op.CompanyID, id)
	if err != nil {
		return nil, errs.NotFound(errs.ErrOrderNotFound)
	}
	payments, err := s.OrderRepo.ListPayments(ctx, op.CompanyID, id)
	if err != nil {
		logger.Warnf("GetOrderDetail: ListPayments failed, orderID=%d, err=%v", id, err)
	}
	refunds, err := s.OrderRepo.ListRefunds(ctx, op.CompanyID, id)
	if err != nil {
		logger.Warnf("GetOrderDetail: ListRefunds failed, orderID=%d, err=%v", id, err)
	}
	logs, err := s.OrderRepo.ListLogs(ctx, op.CompanyID, id)
	if err != nil {
		logger.Warnf("GetOrderDetail: ListLogs failed, orderID=%d, err=%v", id, err)
	}
	delivery, err := s.DeliveryRepo.GetByOrderID(ctx, op.CompanyID, id)
	if err != nil {
		logger.Warnf("GetOrderDetail: GetByOrderID failed, orderID=%d, err=%v", id, err)
	}

	// 允许的下一状态：由领域状态机输出，前端据此渲染阶段推进按钮（单一事实源）
	allowed := domain.OrderAllowedTransitions(o.Status)
	transitions := make([]int, 0, len(allowed))
	for _, st := range allowed {
		transitions = append(transitions, int(st))
	}

	return &dto.OrderDetail{
		Order:              o,
		Payments:           payments,
		Refunds:            refunds,
		Logs:               logs,
		Delivery:           delivery,
		AllowedTransitions: transitions,
	}, nil
}

// CreateOrder 管理端创建订单。
// 校验（#21）：
//  1. 套餐必须属于本公司且为启用状态；
//  2. 客户必须属于本公司（防跨租户挂单）；
//  3. AddonAmount 不得为负（防把订单总额压到极低/负）；
//  4. 金额以"分"计算并由 Total-Deposit 推导 Final（#24），快照同步客户姓名/手机号。
func (s *Service) CreateOrder(ctx context.Context, op Operator, req dto.OrderCreateReq) (*model.Order, error) {
	pkg, err := s.PackageRepo.GetByID(ctx, op.CompanyID, req.PackageID)
	if err != nil {
		return nil, errs.NotFound(errs.ErrPackageNotFound)
	}
	if pkg.Status != enum.PackageStatusActive {
		return nil, errs.BadRequest("套餐已下架，不可下单")
	}
	if req.AddonAmount < 0 {
		return nil, errs.BadRequest("加选金额不能为负")
	}

	// 客户归属校验 + 快照：客户必须属于当前公司
	customerName, customerMobile := "", ""
	if req.CustomerID > 0 {
		cu, err := s.CustomerRepo.GetByID(ctx, op.CompanyID, req.CustomerID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, errs.BadRequest("客户不存在或不属于当前工作室")
			}
			return nil, err
		}
		customerName, customerMobile = cu.Name, cu.Mobile
	}

	// 金额以"分"计算：Deposit + Final == Total 精确相等
	deposit, final, total := domain.SplitOrderAmounts(pkg.BasePrice, pkg.DepositRate, req.AddonAmount)

	o := model.Order{
		TenantBase: model.TenantBase{
			Base:      model.Base{CreatedBy: op.UserID, UpdatedBy: op.UserID},
			CompanyID: op.CompanyID,
		},
		Code:           domain.GenCode("SL"),
		StoreID:        op.StoreID,
		CustomerID:     req.CustomerID,
		CustomerName:   customerName,
		CustomerMobile: customerMobile,
		LeadID:         req.LeadID,
		QuoteID:        req.QuoteID,
		PackageID:      req.PackageID,
		PackageName:    pkg.Name,
		PackageVersion: pkg.Version,
		BasePrice:      pkg.BasePrice,
		AddonAmount:    req.AddonAmount,
		DepositAmt:     deposit,
		FinalAmt:       final,
		TotalAmt:       total,
		ShootDate:      strPtr(req.ShootDate),
		ShootTime:      req.ShootTime,
		ShootAddress:   req.ShootAddress,
		PhotographerID: req.PhotographerID,
		Photographer:   req.Photographer,
		Remark:         req.Remark,
		OwnerID:        orDefaultInt64(req.OwnerID, op.UserID),
		Status:         enum.OrderStatusPendingDeposit,
		PaymentStatus:  enum.PaymentStatusUnpaid,
	}

	err = repository.Tx(func(tx *gorm.DB) error {
		if err := s.OrderRepo.WithTx(tx).Create(ctx, &o); err != nil {
			return err
		}

		if req.CustomerID > 0 {
			if err := s.CustomerRepo.WithTx(tx).Update(ctx, op.CompanyID, req.CustomerID, map[string]interface{}{
				"order_count":  gorm.Expr("order_count + 1"),
				"total_amount": gorm.Expr("total_amount + ?", o.TotalAmt),
			}); err != nil {
				return err
			}
		}

		block := model.CalendarBlock{
			TenantBase: model.TenantBase{
				Base:      model.Base{CreatedBy: op.UserID, UpdatedBy: op.UserID},
				CompanyID: op.CompanyID,
			},
			StoreID:        op.StoreID,
			OrderID:        o.ID,
			CustomerID:     req.CustomerID,
			CustomerName:   customerName,
			Date:           req.ShootDate,
			TimeRange:      req.ShootTime,
			ProjectType:    pkg.Category,
			PhotographerID: req.PhotographerID,
			Photographer:   req.Photographer,
			Status:         enum.BlockStatusLocked,
		}
		if err := s.CalendarRepo.WithTx(tx).Create(ctx, &block); err != nil {
			return err
		}

		if req.LeadID > 0 {
			if err := s.LeadRepo.WithTx(tx).Update(ctx, op.CompanyID, req.LeadID, map[string]interface{}{"status": enum.LeadStatusConfirmed}); err != nil {
				return err
			}
		}
		if req.QuoteID > 0 {
			if err := s.LeadRepo.WithTx(tx).UpdateQuote(ctx, op.CompanyID, req.QuoteID, map[string]interface{}{"status": enum.QuoteStatusConverted}); err != nil {
				return err
			}
		}

		return s.writeOrderLogTx(ctx, tx, o.ID, "create_order", 0, enum.OrderStatusPendingDeposit,
			"创建订单", op)
	})
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (s *Service) UpdateOrder(ctx context.Context, op Operator, id int64, req dto.OrderUpdateReq) error {
	o, err := s.OrderRepo.GetByID(ctx, op.CompanyID, id)
	if err != nil {
		return errs.NotFound(errs.ErrOrderNotFound)
	}
	if o.Status == enum.OrderStatusCompleted || o.Status == enum.OrderStatusCancelled {
		return errs.BadRequest(errs.ErrOrderCompleted)
	}
	return s.OrderRepo.Update(ctx, op.CompanyID, id, map[string]interface{}{
		"shoot_date":      req.ShootDate,
		"shoot_time":      req.ShootTime,
		"shoot_address":   req.ShootAddress,
		"photographer_id": req.PhotographerID,
		"photographer":    req.Photographer,
		"remark":          req.Remark,
		"updated_by":      op.UserID,
	})
}

// ChangeOrderStatus 订单状态流转（完成/取消等终态路径统一走事务）。
// 修复（#18）：四次独立写库（status/finished_at/档期释放/操作日志）收进同一事务，
// 任一步失败整体回滚——避免"订单已取消但档期仍锁定"、"已完成但 finished_at 为空"；
// 事务内加行锁重读订单并二次校验状态机，消除校验与更新间的 TOCTOU。
func (s *Service) ChangeOrderStatus(ctx context.Context, op Operator, id int64, to enum.OrderStatus, content string) error {
	var (
		fromStatus enum.OrderStatus
		customerID int64
		orderCode  string
	)
	err := repository.Tx(func(tx *gorm.DB) error {
		// 1. 事务内行锁读取：基于锁后快照做状态机校验，防并发绕过
		o, err := s.OrderRepo.WithTx(tx).GetByIDForUpdate(ctx, op.CompanyID, id)
		if err != nil {
			return errs.NotFound(errs.ErrOrderNotFound)
		}
		from := o.Status
		fromStatus, customerID, orderCode = from, o.CustomerID, o.Code
		if !domain.OrderCanTransit(from, to) {
			return errs.BadRequest(errs.ErrOrderStatusInvalid)
		}

		updates := map[string]interface{}{"status": to, "updated_by": op.UserID}
		if to == enum.OrderStatusCompleted {
			updates["finished_at"] = time.Now().Format("2006-01-02 15:04:05")
		}
		if err := s.OrderRepo.WithTx(tx).Update(ctx, op.CompanyID, id, updates); err != nil {
			return err
		}

		if to == enum.OrderStatusCancelled {
			// 2. 取消订单同步释放档期锁（同事务，失败即回滚，避免档期无效占用）
			if err := s.OrderRepo.WithTx(tx).UpdateCalendarBlockStatus(ctx, op.CompanyID, id, enum.BlockStatusCancelled); err != nil {
				return err
			}
		}

		// 3. 操作日志
		return s.writeOrderLogTx(ctx, tx, id, "change_status", from, to, content, op)
	})
	if err != nil {
		return err
	}
	// 4. 事务提交后再通知客户：通知失败不能影响已生效的状态流转。
	// 「待确认 → 待拍摄」= 工作室确认了客户的预约，是客户最关心的一个节点。
	if fromStatus == enum.OrderStatusPendingConfirm && to == enum.OrderStatusPendingShoot {
		s.NotifyClient(ctx, op, customerID, "order", "预约已确认",
			"订单 "+orderCode+" 已确认，请按约定时间到店拍摄", "order", id)
	}
	return nil
}

func (s *Service) CancelOrder(ctx context.Context, op Operator, id int64, reason string) error {
	return s.ChangeOrderStatus(ctx, op, id, enum.OrderStatusCancelled, reason)
}
