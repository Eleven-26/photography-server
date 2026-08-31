package service

import (
	"time"

	"gorm.io/gorm"

	"photography-server/internal/model"
	"photography-server/internal/pkg/errs"
)

type RefundCreateReq struct {
	Reason string  `json:"reason"`
	Amount float64 `json:"amount"` // 为空时按规则自动计算
}

// ApplyRefund 申请退款：按拍摄时间距离自动计算可退比例与金额
func (s *Service) ApplyRefund(op Operator, orderID int64, req RefundCreateReq) (*model.OrderRefund, error) {
	var o model.Order
	if err := s.tenant(op).First(&o, orderID).Error; err != nil {
		return nil, errs.NotFound("订单不存在")
	}
	if o.Status == model.OrderStatusCancelled {
		return nil, errs.BadRequest("订单已取消，请直接走已取消流程")
	}
	if o.RefundAmt >= o.PaidAmt {
		return nil, errs.BadRequest("已无可退金额")
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
		return nil, errs.BadRequest("按退款规则当前可退金额为0（距离拍摄不足24小时或未付定金）")
	}

	r := model.OrderRefund{
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
	if err := s.tenant(op).Create(&r).Error; err != nil {
		return nil, err
	}
	s.writeOrderLog(op, orderID, "apply_refund", o.Status, o.Status, "申请退款："+f2(amount)+"（规则档位："+rule+"）")
	return &r, nil
}

func (s *Service) ListRefunds(op Operator, orderID int64) ([]model.OrderRefund, error) {
	var list []model.OrderRefund
	err := s.tenant(op).Where("order_id = ?", orderID).Order("id ASC").Find(&list).Error
	return list, err
}

// AuditRefund 审核退款：通过后累加订单已退金额，订单支付状态置为已退款
func (s *Service) AuditRefund(op Operator, refundID int64, approve bool, remark string) error {
	var r model.OrderRefund
	if err := s.tenant(op).First(&r, refundID).Error; err != nil {
		return errs.NotFound("退款单不存在")
	}
	if r.Status != model.RefundStatusApplying {
		return errs.BadRequest("该退款单已处理")
	}
	var o model.Order
	if err := s.tenant(op).First(&o, r.OrderID).Error; err != nil {
		return errs.NotFound("订单不存在")
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	return s.DB().Transaction(func(tx *gorm.DB) error {
		if approve {
			if err := tx.Model(&r).Updates(map[string]interface{}{
				"status":       model.RefundStatusApproved,
				"audit_by":     op.UserID,
				"audit_at":     now,
				"audit_remark": remark,
			}).Error; err != nil {
				return err
			}
			if err := tx.Model(&o).Updates(map[string]interface{}{
				"refund_amt":     gorm.Expr("refund_amt + ?", r.Amount),
				"payment_status": model.PaymentStatusRefunded,
				"updated_by":     op.UserID,
			}).Error; err != nil {
				return err
			}
			return s.writeOrderLogTx(tx, o.ID, "approve_refund", o.Status, o.Status, "审核通过退款："+f2(r.Amount), op)
		}
		if err := tx.Model(&r).Updates(map[string]interface{}{
			"status":       model.RefundStatusRejected,
			"audit_by":     op.UserID,
			"audit_at":     now,
			"audit_remark": remark,
		}).Error; err != nil {
			return err
		}
		return s.writeOrderLogTx(tx, o.ID, "reject_refund", o.Status, o.Status, "驳回退款申请", op)
	})
}
