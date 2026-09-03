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

func (s *Service) CreateRefund(op Operator, orderID int64, req dto.RefundCreateReq) (*model.OrderRefund, error) {
	o, err := s.OrderRepo.GetByID(op.CompanyID, orderID)
	if err != nil {
		return nil, errs.NotFound(common.ErrOrderNotFound)
	}
	if o.Status == enum.OrderStatusCancelled {
		return nil, errs.BadRequest(common.ErrRefundCancelled)
	}

	amount := req.Amount
	if amount == 0 {
		amount = o.PaidAmt
	}
	if amount <= 0 {
		return nil, errs.BadRequest(common.ErrRefundZero)
	}

	shootDate := ""
	if o.ShootDate != nil {
		shootDate = *o.ShootDate
	}
	shootTime, _ := time.Parse("2006-01-02", shootDate)
	hoursBeforeShoot := time.Until(shootTime).Hours()
	ratio, rule := refundRatio(shootDate, time.Duration(hoursBeforeShoot)*time.Hour)
	refundAmt := round2(amount * ratio)
	if refundAmt <= 0 {
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
		Amount:     refundAmt,
		Reason:     req.Reason,
		RefundRule: rule,
		Status:     enum.RefundStatusApplying,
		ApplyBy:    op.UserID,
		ApplyName:  op.Username,
	}
	if err := s.DB().Create(&rf).Error; err != nil {
		return nil, err
	}
	return &rf, nil
}

func (s *Service) AuditRefund(op Operator, id int64, approved bool, remark string) error {
	rf, err := s.OrderRepo.GetRefundByID(op.CompanyID, id)
	if err != nil {
		return errs.NotFound(common.ErrRefundNotFound)
	}
	if rf.Status != enum.RefundStatusApplying {
		return errs.BadRequest(common.ErrRefundProcessed)
	}

	now := time.Now().Format("2006-01-02 15:04:05")
	status := enum.RefundStatusRejected
	if approved {
		status = enum.RefundStatusApproved
	}

	return s.DB().Transaction(func(tx *gorm.DB) error {
		// 1. 更新退款单状态（CAS：仅当仍为“申请中”时生效，防并发重复审核）
		ok, err := s.OrderRepo.WithTx(tx).AuditRefundApplying(op.CompanyID, id, map[string]interface{}{
			"status":       status,
			"audit_by":     op.UserID,
			"audit_at":     now,
			"audit_remark": remark,
		})
		if err != nil {
			return err
		}
		if !ok {
			return errs.BadRequest(common.ErrRefundProcessed)
		}

		if approved {
			// 2. 事务内锁定读取订单，避免并发审核导致退款金额重复累加
			if _, err := s.OrderRepo.WithTx(tx).GetByIDForUpdate(op.CompanyID, rf.OrderID); err != nil {
				return errs.NotFound(common.ErrOrderNotFound)
			}
			// 3. 累加订单已退金额（带租户过滤，同一事务连接）
			if err := s.OrderRepo.WithTx(tx).Update(op.CompanyID, rf.OrderID, map[string]interface{}{
				"refund_amt":     gorm.Expr("refund_amt + ?", rf.Amount),
				"payment_status": enum.PaymentStatusRefunded,
			}); err != nil {
				return err
			}
			if err := s.OrderRepo.WithTx(tx).UpdateRefund(op.CompanyID, id, map[string]interface{}{"refund_at": now}); err != nil {
				return err
			}
			if err := s.writeOrderLogTx(tx, rf.OrderID, "refund_approved", 0, enum.RefundStatusApproved, "退款审核通过", op); err != nil {
				return err
			}
		} else {
			if err := s.writeOrderLogTx(tx, rf.OrderID, "refund_rejected", 0, enum.RefundStatusRejected, "退款审核驳回: "+remark, op); err != nil {
				return err
			}
		}

		return nil
	})
}

func (s *Service) ListRefunds(op Operator, orderID int64) ([]model.OrderRefund, error) {
	return s.OrderRepo.ListRefunds(op.CompanyID, orderID)
}
