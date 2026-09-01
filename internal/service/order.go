package service

import (
	"errors"
	"strconv"
	"time"

	"gorm.io/gorm"

	"photography-server/internal/common"
	"photography-server/internal/model"
	"photography-server/internal/pkg/errs"
	"photography-server/internal/presentation/dto"
)

func f2(v float64) string {
	return strconv.FormatFloat(v, 'f', 2, 64)
}

// 状态流转白名单：当前状态 -> 允许目标状态
var orderTransition = map[string][]string{
	model.OrderStatusPendingDeposit:   {model.OrderStatusScheduled, model.OrderStatusCancelled},
	model.OrderStatusScheduled:        {model.OrderStatusShooting, model.OrderStatusCancelled},
	model.OrderStatusShooting:         {model.OrderStatusRetouching, model.OrderStatusCancelled},
	model.OrderStatusRetouching:       {model.OrderStatusAwaitingDelivery, model.OrderStatusCancelled},
	model.OrderStatusAwaitingDelivery: {model.OrderStatusCompleted, model.OrderStatusCancelled},
	model.OrderStatusCompleted:        {},
	model.OrderStatusCancelled:        {},
}

func (s *Service) CreateOrder(op Operator, req dto.OrderCreateReq) (*model.Order, error) {
	pkg, err := s.PackageRepo.GetByID(op.CompanyID, req.PackageID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.NotFound(common.ErrPackageNotFound)
		}
		return nil, err
	}
	var customer model.Customer
	if req.CustomerID > 0 {
		c, err := s.CustomerRepo.GetByID(op.CompanyID, req.CustomerID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, errs.NotFound(common.ErrCustomerNotFound)
			}
			return nil, err
		}
		customer = *c
	} else if req.LeadID > 0 {
		c, err := s.ConvertLeadToCustomer(op, req.LeadID)
		if err != nil {
			return nil, err
		}
		customer = *c
	} else {
		return nil, errs.BadRequest(common.ErrOrderNoCustomer)
	}

	deposit := round2(pkg.BasePrice * pkg.DepositRate / 100)
	addon := round2(req.AddonAmount)
	order := model.Order{
		TenantBase: model.TenantBase{
			Base:      model.Base{CreatedBy: op.UserID, UpdatedBy: op.UserID},
			CompanyID: op.CompanyID,
		},
		Code:           genCode("SL"),
		StoreID:        orDefaultInt64(customer.StoreID, op.StoreID),
		CustomerID:     customer.ID,
		CustomerName:   customer.Name,
		CustomerMobile: customer.Mobile,
		LeadID:         req.LeadID,
		QuoteID:        req.QuoteID,
		PackageID:      pkg.ID,
		PackageName:    pkg.Name,
		PackageVersion: pkg.Version,
		BasePrice:      pkg.BasePrice,
		AddonAmount:    addon,
		DepositAmt:     deposit,
		FinalAmt:       round2(pkg.BasePrice - deposit + addon),
		TotalAmt:       round2(pkg.BasePrice + addon),
		Status:         model.OrderStatusPendingDeposit,
		PaymentStatus:  model.PaymentStatusPending,
		ShootDate:      strPtr(req.ShootDate),
		ShootTime:      req.ShootTime,
		ShootAddress:   req.ShootAddress,
		PhotographerID: req.PhotographerID,
		Photographer:   req.Photographer,
		Remark:         req.Remark,
		OwnerID:        orDefaultInt64(req.OwnerID, op.UserID),
	}

	err = s.DB().Transaction(func(tx *gorm.DB) error {
		if err := s.OrderRepo.Create(&order); err != nil {
			return err
		}
		if req.ShootDate != "" {
			block := model.CalendarBlock{
				TenantBase: model.TenantBase{
					Base:      model.Base{CreatedBy: op.UserID, UpdatedBy: op.UserID},
					CompanyID: op.CompanyID,
				},
				StoreID:        order.StoreID,
				OrderID:        order.ID,
				CustomerID:     customer.ID,
				CustomerName:   customer.Name,
				Date:           req.ShootDate,
				TimeRange:      req.ShootTime,
				ProjectType:    pkg.Category,
				PhotographerID: req.PhotographerID,
				Photographer:   req.Photographer,
				Status:         model.BlockStatusLocked,
			}
			if err := s.OrderRepo.CreateCalendarBlock(tx, &block); err != nil {
				return err
			}
		}
		// 线索标记为已成交，报价单标记为已成交
		if req.LeadID > 0 {
			s.LeadRepo.Update(op.CompanyID, req.LeadID, map[string]interface{}{"status": model.LeadStatusConfirmed})
		}
		if req.QuoteID > 0 {
			s.LeadRepo.UpdateQuote(op.CompanyID, req.QuoteID, map[string]interface{}{"status": model.QuoteStatusConverted})
		}
		// 客户冗余统计
		s.CustomerRepo.Update(op.CompanyID, customer.ID, map[string]interface{}{
			"order_count":  gorm.Expr("order_count + 1"),
			"total_amount": gorm.Expr("total_amount + ?", order.TotalAmt),
		})
		return s.writeOrderLogTx(tx, order.ID, "create_order", "", model.OrderStatusPendingDeposit,
			"创建订单，套餐："+order.PackageName+"，总额："+f2(order.TotalAmt), op)
	})
	if err != nil {
		return nil, err
	}
	return &order, nil
}

func (s *Service) ListOrders(op Operator, page, pageSize int, keyword, status string, customerID, photographerID int64, date string) ([]model.Order, int64, error) {
	return s.OrderRepo.List(op.CompanyID, page, pageSize, keyword, status, customerID, photographerID, date)
}

type OrderDetail struct {
	model.Order
	Payments []model.OrderPayment `json:"payments"`
	Refunds  []model.OrderRefund  `json:"refunds"`
	Logs     []model.OrderLog     `json:"logs"`
	Delivery *model.Delivery      `json:"delivery"`
}

func (s *Service) GetOrder(op Operator, id int64) (*OrderDetail, error) {
	o, err := s.OrderRepo.GetByID(op.CompanyID, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.NotFound(common.ErrOrderNotFound)
		}
		return nil, err
	}
	d := &OrderDetail{Order: *o}
	d.Payments, _ = s.OrderRepo.ListPayments(op.CompanyID, id)
	d.Refunds, _ = s.OrderRepo.ListRefunds(op.CompanyID, id)
	d.Logs, _ = s.OrderRepo.ListLogs(op.CompanyID, id)
	dv, err := s.OrderRepo.GetDelivery(op.CompanyID, id)
	if err == nil {
		d.Delivery = dv
	}
	return d, nil
}

func (s *Service) UpdateOrder(op Operator, id int64, req dto.OrderUpdateReq) error {
	o, err := s.OrderRepo.GetByID(op.CompanyID, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errs.NotFound(common.ErrOrderNotFound)
		}
		return err
	}
	if o.Status == model.OrderStatusCompleted || o.Status == model.OrderStatusCancelled {
		return errs.BadRequest(common.ErrOrderCompleted)
	}
	err = s.DB().Transaction(func(tx *gorm.DB) error {
		if err := s.OrderRepo.Update(op.CompanyID, id, map[string]interface{}{
			"shoot_date": req.ShootDate, "shoot_time": req.ShootTime,
			"shoot_address": req.ShootAddress, "photographer_id": req.PhotographerID,
			"photographer": req.Photographer, "remark": req.Remark, "updated_by": op.UserID,
		}); err != nil {
			return err
		}
		// 同步更新关联档期
		return s.OrderRepo.UpdateCalendarBlockByOrder(tx, id, map[string]interface{}{
			"date": req.ShootDate, "time_range": req.ShootTime,
			"photographer_id": req.PhotographerID, "photographer": req.Photographer,
			"updated_by": op.UserID,
		})
	})
	return err
}

// ChangeOrderStatus 订单状态流转（含档期释放、完成时间等联动）
func (s *Service) ChangeOrderStatus(op Operator, id int64, to string, content string) error {
	o, err := s.OrderRepo.GetByID(op.CompanyID, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errs.NotFound(common.ErrOrderNotFound)
		}
		return err
	}
	allowed, ok := orderTransition[o.Status]
	if !ok {
		return errs.BadRequest(common.ErrOrderStatusInvalid)
	}
	valid := false
	for _, s := range allowed {
		if s == to {
			valid = true
			break
		}
	}
	if !valid {
		return errs.BadRequest(common.ErrOrderStatusInvalid)
	}

	err = s.DB().Transaction(func(tx *gorm.DB) error {
		updates := map[string]interface{}{"status": to, "updated_by": op.UserID}
		if to == model.OrderStatusCompleted {
			now := time.Now().Format("2006-01-02 15:04:05")
			updates["finished_at"] = now
		}
		if to == model.OrderStatusCancelled {
			updates["cancel_reason"] = content
			// 释放档期
			if err := s.OrderRepo.CancelCalendarBlockByOrder(tx, id); err != nil {
				return err
			}
		}
		if err := s.OrderRepo.Update(op.CompanyID, id, updates); err != nil {
			return err
		}
		return s.writeOrderLogTx(tx, id, "change_status", o.Status, to, content, op)
	})
	return err
}

func (s *Service) CancelOrder(op Operator, id int64, reason string) error {
	return s.ChangeOrderStatus(op, id, model.OrderStatusCancelled, reason)
}

// writeOrderLogTx 记录订单操作日志
func (s *Service) writeOrderLogTx(tx *gorm.DB, orderID int64, action, from, to, content string, op Operator) error {
	return s.OrderRepo.CreateLog(tx, &model.OrderLog{
		TenantBase: model.TenantBase{
			Base:      model.Base{CreatedBy: op.UserID, UpdatedBy: op.UserID},
			CompanyID: op.CompanyID,
		},
		OrderID:      orderID,
		Action:       action,
		FromStatus:   from,
		ToStatus:     to,
		Content:      content,
		OperatorID:   op.UserID,
		OperatorName: op.Nickname,
	})
}
