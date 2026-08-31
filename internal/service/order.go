package service

import (
	"strconv"
	"time"

	"gorm.io/gorm"

	"photography-server/internal/model"
	"photography-server/internal/pkg/errs"
)

func f2(v float64) string {
	return strconv.FormatFloat(v, 'f', 2, 64)
}

type OrderCreateReq struct {
	CustomerID     int64   `json:"customer_id"`
	LeadID         int64   `json:"lead_id"`
	QuoteID        int64   `json:"quote_id"`
	PackageID      int64   `json:"package_id" binding:"required"`
	AddonAmount    float64 `json:"addon_amount"`
	ShootDate      string  `json:"shoot_date"`
	ShootTime      string  `json:"shoot_time"`
	ShootAddress   string  `json:"shoot_address"`
	PhotographerID int64   `json:"photographer_id"`
	Photographer   string  `json:"photographer"`
	Remark         string  `json:"remark"`
	OwnerID        int64   `json:"owner_id"`
}

type OrderUpdateReq struct {
	ShootDate      string `json:"shoot_date"`
	ShootTime      string `json:"shoot_time"`
	ShootAddress   string `json:"shoot_address"`
	PhotographerID int64  `json:"photographer_id"`
	Photographer   string `json:"photographer"`
	Remark         string `json:"remark"`
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

func (s *Service) CreateOrder(op Operator, req OrderCreateReq) (*model.Order, error) {
	var pkg model.Package
	if err := s.tenant(op).First(&pkg, req.PackageID).Error; err != nil {
		return nil, errs.NotFound("套餐不存在")
	}
	var customer model.Customer
	if req.CustomerID > 0 {
		if err := s.tenant(op).First(&customer, req.CustomerID).Error; err != nil {
			return nil, errs.NotFound("客户不存在")
		}
	} else if req.LeadID > 0 {
		c, err := s.ConvertLeadToCustomer(op, req.LeadID)
		if err != nil {
			return nil, err
		}
		customer = *c
	} else {
		return nil, errs.BadRequest("请选择客户或来源线索")
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

	err := s.DB().Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&order).Error; err != nil {
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
			if err := tx.Create(&block).Error; err != nil {
				return err
			}
		}
		// 线索标记为已成交，报价单标记为已成交
		if req.LeadID > 0 {
			tx.Model(&model.Lead{}).Where("id = ?", req.LeadID).Update("status", model.LeadStatusConfirmed)
		}
		if req.QuoteID > 0 {
			tx.Model(&model.Quote{}).Where("id = ?", req.QuoteID).Update("status", model.QuoteStatusConverted)
		}
		// 客户冗余统计
		tx.Model(&model.Customer{}).Where("id = ?", customer.ID).Updates(map[string]interface{}{
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
	q := s.tenant(op)
	if keyword != "" {
		kw := "%" + keyword + "%"
		q = q.Where("code LIKE ? OR customer_name LIKE ? OR customer_mobile LIKE ?", kw, kw, kw)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if customerID > 0 {
		q = q.Where("customer_id = ?", customerID)
	}
	if photographerID > 0 {
		q = q.Where("photographer_id = ?", photographerID)
	}
	if date != "" {
		q = q.Where("shoot_date = ?", date)
	}
	var total int64
	if err := q.Model(&model.Order{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.Order
	page, pageSize = normalizePage(page, pageSize)
	if err := q.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

type OrderDetail struct {
	model.Order
	Payments []model.OrderPayment `json:"payments"`
	Refunds  []model.OrderRefund  `json:"refunds"`
	Logs     []model.OrderLog     `json:"logs"`
	Delivery *model.Delivery      `json:"delivery"`
}

func (s *Service) GetOrder(op Operator, id int64) (*OrderDetail, error) {
	var o model.Order
	if err := s.tenant(op).First(&o, id).Error; err != nil {
		return nil, errs.NotFound("订单不存在")
	}
	d := &OrderDetail{Order: o}
	s.tenant(op).Where("order_id = ?", id).Order("id ASC").Find(&d.Payments)
	s.tenant(op).Where("order_id = ?", id).Order("id ASC").Find(&d.Refunds)
	s.tenant(op).Where("order_id = ?", id).Order("id ASC").Find(&d.Logs)
	var dv model.Delivery
	if err := s.tenant(op).Where("order_id = ?", id).First(&dv).Error; err == nil {
		d.Delivery = &dv
	}
	return d, nil
}

func (s *Service) UpdateOrder(op Operator, id int64, req OrderUpdateReq) error {
	var o model.Order
	if err := s.tenant(op).First(&o, id).Error; err != nil {
		return errs.NotFound("订单不存在")
	}
	if o.Status == model.OrderStatusCompleted || o.Status == model.OrderStatusCancelled {
		return errs.BadRequest("已完成或已取消的订单不可修改")
	}
	err := s.DB().Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&o).Updates(map[string]interface{}{
			"shoot_date": req.ShootDate, "shoot_time": req.ShootTime,
			"shoot_address": req.ShootAddress, "photographer_id": req.PhotographerID,
			"photographer": req.Photographer, "remark": req.Remark, "updated_by": op.UserID,
		}).Error; err != nil {
			return err
		}
		// 同步更新关联档期
		return tx.Model(&model.CalendarBlock{}).Where("order_id = ?", id).Updates(map[string]interface{}{
			"date": req.ShootDate, "time_range": req.ShootTime,
			"photographer_id": req.PhotographerID, "photographer": req.Photographer,
			"updated_by": op.UserID,
		}).Error
	})
	return err
}

// ChangeOrderStatus 订单状态流转（含档期释放、完成时间等联动）
func (s *Service) ChangeOrderStatus(op Operator, id int64, to string, content string) error {
	var o model.Order
	if err := s.tenant(op).First(&o, id).Error; err != nil {
		return errs.NotFound("订单不存在")
	}
	allowed, ok := orderTransition[o.Status]
	if !ok {
		return errs.BadRequest("订单状态不允许流转")
	}
	valid := false
	for _, s := range allowed {
		if s == to {
			valid = true
			break
		}
	}
	if !valid {
		return errs.BadRequest("订单状态不允许从 " + o.Status + " 流转到 " + to)
	}

	err := s.DB().Transaction(func(tx *gorm.DB) error {
		updates := map[string]interface{}{"status": to, "updated_by": op.UserID}
		if to == model.OrderStatusCompleted {
			now := time.Now().Format("2006-01-02 15:04:05")
			updates["finished_at"] = now
		}
		if to == model.OrderStatusCancelled {
			updates["cancel_reason"] = content
			// 释放档期
			if err := tx.Model(&model.CalendarBlock{}).Where("order_id = ?", id).
				Update("status", model.BlockStatusCancelled).Error; err != nil {
				return err
			}
		}
		if err := tx.Model(&o).Updates(updates).Error; err != nil {
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
	return tx.Create(&model.OrderLog{
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
	}).Error
}
