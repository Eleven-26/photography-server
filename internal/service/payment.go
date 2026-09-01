package service

import (
	"errors"
	"time"

	"gorm.io/gorm"

	"photography-server/internal/common"
	"photography-server/internal/model"
	"photography-server/internal/pkg/errs"
	"photography-server/internal/presentation/dto"
)

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

func (s *Service) CreatePayment(op Operator, orderID int64, req dto.PaymentCreateReq) (*model.OrderPayment, error) {
	o, err := s.OrderRepo.GetByID(op.CompanyID, orderID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.NotFound(common.ErrOrderNotFound)
		}
		return nil, err
	}
	methodName := "其他"
	if req.MethodID > 0 {
		method, err := s.SettingsRepo.GetPaymentMethodByID(op.CompanyID, req.MethodID)
		if err == nil {
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
	if err := s.OrderRepo.CreatePayment(&p); err != nil {
		return nil, err
	}
	s.writeOrderLog(op, orderID, "create_payment", o.Status, o.Status, "登记收款("+typeName(req.Type)+")："+f2(p.Amount))
	return &p, nil
}

func (s *Service) ListPayments(op Operator, orderID int64) ([]model.OrderPayment, error) {
	return s.OrderRepo.ListPayments(op.CompanyID, orderID)
}

// ConfirmPayment 核验收款：累加已收金额，定金到账后订单进入待拍摄
func (s *Service) ConfirmPayment(op Operator, paymentID int64) error {
	p, err := s.OrderRepo.GetPaymentByID(op.CompanyID, paymentID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errs.NotFound(common.ErrPaymentNotFound)
		}
		return err
	}
	if p.Status == model.PaymentStatusConfirmed {
		return errs.BadRequest(common.ErrPaymentConfirmed)
	}
	o, err := s.OrderRepo.GetByID(op.CompanyID, p.OrderID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errs.NotFound(common.ErrOrderNotFound)
		}
		return err
	}

	err = s.DB().Transaction(func(tx *gorm.DB) error {
		if err := s.OrderRepo.UpdatePayment(op.CompanyID, paymentID, map[string]interface{}{
			"status":     model.PaymentStatusConfirmed,
			"paid_at":    time.Now().Format("2006-01-02 15:04:05"),
			"updated_by": op.UserID,
		}); err != nil {
			return err
		}
		// 订单累计已收
		updates := map[string]interface{}{
			"paid_amt":       gorm.Expr("paid_amt + ?", p.Amount),
			"payment_status": model.PaymentStatusConfirmed,
			"updated_by":     op.UserID,
		}
		if err := s.OrderRepo.Update(op.CompanyID, p.OrderID, updates); err != nil {
			return err
		}
		// 定金到账：待定金 -> 待拍摄
		if p.Type == "deposit" && o.Status == model.OrderStatusPendingDeposit {
			if err := s.OrderRepo.Update(op.CompanyID, p.OrderID, map[string]interface{}{"status": model.OrderStatusScheduled}); err != nil {
				return err
			}
		}
		// 尾款/加选结清：待交付 -> 已完成（若已待交付则直接完成）
		if (p.Type == "final" || p.Type == "addon") && o.Status == model.OrderStatusAwaitingDelivery {
			paid := o.PaidAmt + p.Amount
			if paid >= o.TotalAmt {
				now := time.Now().Format("2006-01-02 15:04:05")
				s.OrderRepo.Update(op.CompanyID, p.OrderID, map[string]interface{}{"status": model.OrderStatusCompleted, "finished_at": now})
			}
		}
		return s.writeOrderLogTx(tx, o.ID, "confirm_payment", o.Status, o.Status, "核验收款("+typeName(p.Type)+")："+f2(p.Amount), op)
	})
	return err
}

// DeletePayment 删除收款记录（已核验的先回退金额）
func (s *Service) DeletePayment(op Operator, paymentID int64) error {
	p, err := s.OrderRepo.GetPaymentByID(op.CompanyID, paymentID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errs.NotFound(common.ErrPaymentNotFound)
		}
		return err
	}
	return s.DB().Transaction(func(tx *gorm.DB) error {
		if p.Status == model.PaymentStatusConfirmed {
			if err := s.OrderRepo.Update(op.CompanyID, p.OrderID, map[string]interface{}{
				"paid_amt": gorm.Expr("paid_amt - ?", p.Amount),
			}); err != nil {
				return err
			}
		}
		return s.OrderRepo.DeletePayment(tx, paymentID)
	})
}

func (s *Service) writeOrderLog(op Operator, orderID int64, action, from, to, content string) error {
	return s.writeOrderLogTx(s.DB(), orderID, action, from, to, content, op)
}
