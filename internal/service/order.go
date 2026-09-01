package service

import (
	"time"

	"gorm.io/gorm"

	"photography-server/internal/common"
	"photography-server/internal/enum"
	"photography-server/internal/model"
	"photography-server/internal/pkg/errs"
	"photography-server/internal/presentation/dto"
)

// orderAllowedTransitions 定义允许的订单状态流转
var orderAllowedTransitions = map[enum.OrderStatus][]enum.OrderStatus{
	enum.OrderStatusPendingDeposit:  {enum.OrderStatusPendingShoot, enum.OrderStatusCancelled},
	enum.OrderStatusPendingShoot:    {enum.OrderStatusShooting, enum.OrderStatusCancelled},
	enum.OrderStatusShooting:        {enum.OrderStatusRetouching, enum.OrderStatusCancelled},
	enum.OrderStatusRetouching:      {enum.OrderStatusPendingDelivery, enum.OrderStatusCancelled},
	enum.OrderStatusPendingDelivery: {enum.OrderStatusCompleted, enum.OrderStatusCancelled},
	enum.OrderStatusCompleted:       {},
	enum.OrderStatusCancelled:       {},
}

func (s *Service) ListOrders(op Operator, page, pageSize int, status string, customerID int64) ([]model.Order, int64, error) {
	return s.OrderRepo.List(op.CompanyID, page, pageSize, status, customerID)
}

func (s *Service) GetOrderDetail(op Operator, id int64) (*dto.OrderDetail, error) {
	o, err := s.OrderRepo.GetByID(op.CompanyID, id)
	if err != nil {
		return nil, errs.NotFound(common.ErrOrderNotFound)
	}
	payments, _ := s.OrderRepo.ListPayments(op.CompanyID, id)
	refunds, _ := s.OrderRepo.ListRefunds(op.CompanyID, id)
	logs, _ := s.OrderRepo.ListLogs(op.CompanyID, id)
	delivery, _ := s.DeliveryRepo.GetByOrderID(op.CompanyID, id)

	return &dto.OrderDetail{
		Order:    o,
		Payments: payments,
		Refunds:  refunds,
		Logs:     logs,
		Delivery: delivery,
	}, nil
}

func (s *Service) CreateOrder(op Operator, req dto.OrderCreateReq) (*model.Order, error) {
	var pkg model.Package
	if err := s.DB().Where("id = ?", req.PackageID).First(&pkg).Error; err != nil {
		return nil, errs.NotFound(common.ErrPackageNotFound)
	}

	o := model.Order{
		TenantBase: model.TenantBase{
			Base:      model.Base{CreatedBy: op.UserID, UpdatedBy: op.UserID},
			CompanyID: op.CompanyID,
		},
		Code:           genCode("SL"),
		StoreID:        op.StoreID,
		CustomerID:     req.CustomerID,
		QuoteID:        req.QuoteID,
		PackageID:      req.PackageID,
		PackageName:    pkg.Name,
		PackageVersion: pkg.Version,
		BasePrice:      pkg.BasePrice,
		AddonAmount:    req.AddonAmount,
		DepositAmt:     round2(pkg.BasePrice * pkg.DepositRate / 100),
		FinalAmt:       round2(pkg.BasePrice - pkg.BasePrice*pkg.DepositRate/100 + req.AddonAmount),
		TotalAmt:       round2(pkg.BasePrice + req.AddonAmount),
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

	err := s.DB().Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&o).Error; err != nil {
			return err
		}

		if req.CustomerID > 0 {
			s.DB().Model(&model.Customer{}).Where("id = ?", req.CustomerID).
				Updates(map[string]interface{}{"order_count": gorm.Expr("order_count + 1"), "total_amount": gorm.Expr("total_amount + ?", o.TotalAmt)})
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
		if err := tx.Create(&block).Error; err != nil {
			return err
		}

		if req.LeadID > 0 {
			s.LeadRepo.Update(op.CompanyID, req.LeadID, map[string]interface{}{"status": enum.LeadStatusConfirmed})
		}
		if req.QuoteID > 0 {
			s.LeadRepo.UpdateQuote(op.CompanyID, req.QuoteID, map[string]interface{}{"status": enum.QuoteStatusConverted})
		}

		return s.writeOrderLogTx(tx, o.ID, "create_order", 0, enum.OrderStatusPendingDeposit,
			"创建订单", op)
	})
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (s *Service) UpdateOrder(op Operator, id int64, req dto.OrderUpdateReq) error {
	o, err := s.OrderRepo.GetByID(op.CompanyID, id)
	if err != nil {
		return errs.NotFound(common.ErrOrderNotFound)
	}
	if o.Status == enum.OrderStatusCompleted || o.Status == enum.OrderStatusCancelled {
		return errs.BadRequest(common.ErrOrderCompleted)
	}
	return s.OrderRepo.Update(op.CompanyID, id, map[string]interface{}{
		"shoot_date":      req.ShootDate,
		"shoot_time":      req.ShootTime,
		"shoot_address":   req.ShootAddress,
		"photographer_id": req.PhotographerID,
		"photographer":    req.Photographer,
		"remark":          req.Remark,
		"updated_by":      op.UserID,
	})
}

func (s *Service) ChangeOrderStatus(op Operator, id int64, to enum.OrderStatus, content string) error {
	o, err := s.OrderRepo.GetByID(op.CompanyID, id)
	if err != nil {
		return errs.NotFound(common.ErrOrderNotFound)
	}
	from := o.Status
	allowed := orderAllowedTransitions[from]
	found := false
	for _, a := range allowed {
		if a == to {
			found = true
			break
		}
	}
	if !found {
		return errs.BadRequest(common.ErrOrderStatusInvalid)
	}

	if err := s.OrderRepo.Update(op.CompanyID, id, map[string]interface{}{"status": to, "updated_by": op.UserID}); err != nil {
		return err
	}

	if to == enum.OrderStatusCompleted {
		now := time.Now().Format("2006-01-02 15:04:05")
		s.OrderRepo.Update(op.CompanyID, id, map[string]interface{}{"finished_at": now})
	}
	if to == enum.OrderStatusCancelled {
		s.OrderRepo.UpdateCalendarBlockStatus(op.CompanyID, id, enum.BlockStatusCancelled)
	}

	return s.writeOrderLog(id, "change_status", from, to, content, op)
}

func (s *Service) CancelOrder(op Operator, id int64, reason string) error {
	return s.ChangeOrderStatus(op, id, enum.OrderStatusCancelled, reason)
}
