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

// ApplyRefund 申请退款：按拍摄时间距离自动计算可退比例与金额
func (s *Service) ApplyRefund(op Operator, orderID int64, req dto.RefundCreateReq) (*model.OrderRefund, error) {
	o, err := s.OrderRepo.GetByID(op.CompanyID, orderID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.NotFound(common.ErrOrderNotFound)
		}
		return nil, err
	}
	if o.Status == model.OrderStatusCancelled {
		return nil, errs.BadRequest(common.ErrRefundCancelled)
	}
	if o.RefundAmt >= o.PaidAmt {
		return nil, errs.BadRequest(common.ErrRefundZero)
	}

	ratio := 0.0
	rule := ""
	if o.ShootDate != nil && *o.ShootDate != "" {
		shoot, err := time.ParseInLocation("2006-01-02", *o.ShootDate, time.Local)
		if err == nil {
			diff := time.Until(shoot)
			if diff < 0 {
				ratio, rule = 0, "shoot_past"
			} else {
				ratio, rule = refundRatio(*o.ShootDate, diff)
			}
		}
	} else {
		ratio, rule = 0, "no_shoot_date"
	}

	// 可退基数=已收定金部分（封顶 order.deposit_amt）
	base := o.DepositAmt
	if o.PaidAmt < base {
		base = o.PaidAmt
	}
	amount := round2(base * ratio)
	if req.Amount > 0 && req.Amount < amount {
		amount = round2(req.Amount)
	}
	if amount <= 0 {
		return nil, errs.BadRequest(common.ErrRefundNoTime)
	}

	rf := model.OrderRefund{
		TenantBase: model.TenantBase{
			Base:      model.Base{CreatedBy: op.UserID, UpdatedBy: op.UserID},
			CompanyID: op.CompanyID,
		},
		OrderID:    orderID,
		Code:       genCode("RF"),
		CustomerID: o.CustomerID,
		Amount:     amount,
		Reason:     req.Reason,
		RefundRule: rule,
		Status:     model.RefundStatusApplying,
		ApplyBy:    op.UserID,
		ApplyName:  op.Nickname,
	}
	if err := s.OrderRepo.CreateRefund(&rf); err != nil {
		return nil, err
	}
	s.writeOrderLog(op, orderID, "apply_refund", o.Status, o.Status, "申请退款："+f2(amount)+"（规则档位："+rule+"）")
	return &rf, nil
}

func (s *Service) ListRefunds(op Operator, orderID int64) ([]model.OrderRefund, error) {
	return s.OrderRepo.ListRefunds(op.CompanyID, orderID)
}

// AuditRefund 审核退款：通过后累加订单已退金额，订单支付状态置为已退款
func (s *Service) AuditRefund(op Operator, refundID int64, approve bool, remark string) error {
	rf, err := s.OrderRepo.GetRefundByID(op.CompanyID, refundID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errs.NotFound(common.ErrRefundNotFound)
		}
		return err
	}
	if rf.Status != model.RefundStatusApplying {
		return errs.BadRequest(common.ErrRefundProcessed)
	}
	o, err := s.OrderRepo.GetByID(op.CompanyID, rf.OrderID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errs.NotFound(common.ErrOrderNotFound)
		}
		return err
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	return s.DB().Transaction(func(tx *gorm.DB) error {
		if approve {
			if err := s.OrderRepo.UpdateRefund(op.CompanyID, refundID, map[string]interface{}{
				"status":       model.RefundStatusApproved,
				"audit_by":     op.UserID,
				"audit_at":     now,
				"audit_remark": remark,
			}); err != nil {
				return err
			}
			if err := s.OrderRepo.Update(op.CompanyID, rf.OrderID, map[string]interface{}{
				"refund_amt":     gorm.Expr("refund_amt + ?", rf.Amount),
				"payment_status": model.PaymentStatusRefunded,
				"updated_by":     op.UserID,
			}); err != nil {
				return err
			}
			return s.writeOrderLogTx(tx, o.ID, "approve_refund", o.Status, o.Status, "审核通过退款："+f2(rf.Amount), op)
		}
		if err := s.OrderRepo.UpdateRefund(op.CompanyID, refundID, map[string]interface{}{
			"status":       model.RefundStatusRejected,
			"audit_by":     op.UserID,
			"audit_at":     now,
			"audit_remark": remark,
		}); err != nil {
			return err
		}
		return s.writeOrderLogTx(tx, o.ID, "reject_refund", o.Status, o.Status, "驳回退款申请", op)
	})
}
