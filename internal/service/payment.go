package service

import (
	"time"

	"gorm.io/gorm"

	"photography-server/internal/model"
	"photography-server/internal/pkg/errs"
)

type PaymentCreateReq struct {
	Type     string  `json:"type" binding:"required"` // deposit-定金 final-尾款 addon-加选
	Amount   float64 `json:"amount" binding:"required"`
	MethodID int64   `json:"method_id"`
	PaidAt   string  `json:"paid_at"`
	Voucher  string  `json:"voucher"`
	Remark   string  `json:"remark"`
}

func (s *Service) CreatePayment(op Operator, orderID int64, req PaymentCreateReq) (*model.OrderPayment, error) {
	var o model.Order
	if err := s.tenant(op).First(&o, orderID).Error; err != nil {
		return nil, errs.NotFound("订单不存在")
	}
	methodName := "其他"
	var method model.PaymentMethod
	if req.MethodID > 0 {
		if err := s.tenant(op).First(&method, req.MethodID).Error; err == nil {
			methodName = method.Name
		}
	}
	p := model.OrderPayment{
		TenantBase: model.TenantBase{
			Base:      model.Base{CreatedBy: op.UserID, UpdatedBy: op.UserID},
			CompanyID: op.CompanyID,
		},
		OrderID:      orderID,
		Code:         genCode("PM"),
		CustomerID:   o.CustomerID,
		Type:         req.Type,
		Amount:       round2(req.Amount),
		MethodID:     req.MethodID,
		MethodName:   methodName,
		Status:       model.PaymentStatusPending,
		PaidAt:       strPtr(req.PaidAt),
		Voucher:      req.Voucher,
		OperatorID:   op.UserID,
		OperatorName: op.Nickname,
		Remark:       req.Remark,
	}
	if err := s.tenant(op).Create(&p).Error; err != nil {
		return nil, err
	}
	s.writeOrderLog(op, orderID, "create_payment", o.Status, o.Status, "登记收款("+typeName(req.Type)+")："+f2(p.Amount))
	return &p, nil
}

func typeName(t string) string {
	switch t {
	case "deposit":
		return "定金"
	case "final":
		return "尾款"
	case "addon":
		return "加选"
	}
	return t
}

func (s *Service) ListPayments(op Operator, orderID int64) ([]model.OrderPayment, error) {
	var list []model.OrderPayment
	err := s.tenant(op).Where("order_id = ?", orderID).Order("id ASC").Find(&list).Error
	return list, err
}

// ConfirmPayment 核验收款：累加已收金额，定金到账后订单进入待拍摄
func (s *Service) ConfirmPayment(op Operator, paymentID int64) error {
	var p model.OrderPayment
	if err := s.tenant(op).First(&p, paymentID).Error; err != nil {
		return errs.NotFound("收款记录不存在")
	}
	if p.Status == model.PaymentStatusConfirmed {
		return errs.BadRequest("该收款已核验")
	}
	var o model.Order
	if err := s.tenant(op).First(&o, p.OrderID).Error; err != nil {
		return errs.NotFound("订单不存在")
	}

	err := s.DB().Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&p).Updates(map[string]interface{}{
			"status":     model.PaymentStatusConfirmed,
			"paid_at":    time.Now().Format("2006-01-02 15:04:05"),
			"updated_by": op.UserID,
		}).Error; err != nil {
			return err
		}
		// 订单累计已收
		updates := map[string]interface{}{
			"paid_amt":       gorm.Expr("paid_amt + ?", p.Amount),
			"payment_status": model.PaymentStatusConfirmed,
			"updated_by":     op.UserID,
		}
		if err := tx.Model(&o).Updates(updates).Error; err != nil {
			return err
		}
		// 定金到账：待定金 -> 待拍摄
		if p.Type == "deposit" && o.Status == model.OrderStatusPendingDeposit {
			if err := tx.Model(&o).Update("status", model.OrderStatusScheduled).Error; err != nil {
				return err
			}
		}
		// 尾款/加选结清：待交付 -> 已完成（若已待交付则直接完成）
		if (p.Type == "final" || p.Type == "addon") && o.Status == model.OrderStatusAwaitingDelivery {
			paid := o.PaidAmt + p.Amount
			if paid >= o.TotalAmt {
				now := time.Now().Format("2006-01-02 15:04:05")
				tx.Model(&o).Updates(map[string]interface{}{"status": model.OrderStatusCompleted, "finished_at": now})
			}
		}
		return s.writeOrderLogTx(tx, o.ID, "confirm_payment", o.Status, o.Status, "核验收款("+typeName(p.Type)+")："+f2(p.Amount), op)
	})
	return err
}

// DeletePayment 删除收款记录（已核验的先回退金额）
func (s *Service) DeletePayment(op Operator, paymentID int64) error {
	var p model.OrderPayment
	if err := s.tenant(op).First(&p, paymentID).Error; err != nil {
		return errs.NotFound("收款记录不存在")
	}
	return s.DB().Transaction(func(tx *gorm.DB) error {
		if p.Status == model.PaymentStatusConfirmed {
			if err := tx.Model(&model.Order{}).Where("id = ?", p.OrderID).
				Update("paid_amt", gorm.Expr("paid_amt - ?", p.Amount)).Error; err != nil {
				return err
			}
		}
		return tx.Delete(&p).Error
	})
}

func (s *Service) writeOrderLog(op Operator, orderID int64, action, from, to, content string) error {
	return s.writeOrderLogTx(s.tenant(op), orderID, action, from, to, content, op)
}
