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

func (s *Service) ListOrders(ctx context.Context, op Operator, page, pageSize int, status string, customerID int64) ([]model.Order, int64, error) {
	return s.OrderRepo.List(ctx, op.CompanyID, page, pageSize, status, customerID)
}

func (s *Service) GetOrderDetail(ctx context.Context, op Operator, id int64) (*dto.OrderDetail, error) {
	o, err := s.OrderRepo.GetByID(ctx, op.CompanyID, id)
	if err != nil {
		return nil, errs.NotFound(errs.ErrOrderNotFound)
	}
	payments, _ := s.OrderRepo.ListPayments(ctx, op.CompanyID, id)
	refunds, _ := s.OrderRepo.ListRefunds(ctx, op.CompanyID, id)
	logs, _ := s.OrderRepo.ListLogs(ctx, op.CompanyID, id)
	delivery, _ := s.DeliveryRepo.GetByOrderID(ctx, op.CompanyID, id)

	return &dto.OrderDetail{
		Order:    o,
		Payments: payments,
		Refunds:  refunds,
		Logs:     logs,
		Delivery: delivery,
	}, nil
}

func (s *Service) CreateOrder(ctx context.Context, op Operator, req dto.OrderCreateReq) (*model.Order, error) {
	pkg, err := s.PackageRepo.GetByID(ctx, op.CompanyID, req.PackageID)
	if err != nil {
		return nil, errs.NotFound(errs.ErrPackageNotFound)
	}

	o := model.Order{
		TenantBase: model.TenantBase{
			Base:      model.Base{CreatedBy: op.UserID, UpdatedBy: op.UserID},
			CompanyID: op.CompanyID,
		},
		Code:           domain.GenCode("SL"),
		StoreID:        op.StoreID,
		CustomerID:     req.CustomerID,
		QuoteID:        req.QuoteID,
		PackageID:      req.PackageID,
		PackageName:    pkg.Name,
		PackageVersion: pkg.Version,
		BasePrice:      pkg.BasePrice,
		AddonAmount:    req.AddonAmount,
		DepositAmt:     domain.Round2(pkg.BasePrice * pkg.DepositRate / 100),
		FinalAmt:       domain.Round2(pkg.BasePrice - pkg.BasePrice*pkg.DepositRate/100 + req.AddonAmount),
		TotalAmt:       domain.Round2(pkg.BasePrice + req.AddonAmount),
		ShootDate:      strPtr(req.ShootDate),
		ShootTime:      req.ShootTime,
		ShootAddress:   req.ShootAddress,
		PhotographerID: req.PhotographerID,
		Photographer:   req.Photographer,
		Remark:         req.Remark,
		OwnerID:        orDefaultInt64(req.OwnerID, op.UserID),
		Status:         enum.OrderStatusPendingDeposit,
		PaymentStatus:  enum.PaymentStatusPending,
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
			CustomerName:   "",
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

func (s *Service) ChangeOrderStatus(ctx context.Context, op Operator, id int64, to enum.OrderStatus, content string) error {
	o, err := s.OrderRepo.GetByID(ctx, op.CompanyID, id)
	if err != nil {
		return errs.NotFound(errs.ErrOrderNotFound)
	}
	from := o.Status
	if !domain.OrderCanTransit(from, to) {
		return errs.BadRequest(errs.ErrOrderStatusInvalid)
	}

	if err := s.OrderRepo.Update(ctx, op.CompanyID, id, map[string]interface{}{"status": to, "updated_by": op.UserID}); err != nil {
		return err
	}

	if to == enum.OrderStatusCompleted {
		now := time.Now().Format("2006-01-02 15:04:05")
		s.OrderRepo.Update(ctx, op.CompanyID, id, map[string]interface{}{"finished_at": now})
	}
	if to == enum.OrderStatusCancelled {
		s.OrderRepo.UpdateCalendarBlockStatus(ctx, op.CompanyID, id, enum.BlockStatusCancelled)
	}

	return s.writeOrderLog(ctx, id, "change_status", from, to, content, op)
}

func (s *Service) CancelOrder(ctx context.Context, op Operator, id int64, reason string) error {
	return s.ChangeOrderStatus(ctx, op, id, enum.OrderStatusCancelled, reason)
}
