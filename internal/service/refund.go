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
		if err := tx.Model(&rf).Updates(map[string]interface{}{
			"status":       status,
			"audit_by":     op.UserID,
			"audit_at":     now,
			"audit_remark": remark,
		}).Error; err != nil {
			return err
		}

		if approved {
			if err := tx.Model(&model.Order{}).Where("id = ?", rf.OrderID).Updates(map[string]interface{}{
				"refund_amt":     gorm.Expr("refund_amt + ?", rf.Amount),
				"payment_status": enum.PaymentStatusRefunded,
			}).Error; err != nil {
				return err
			}
			tx.Model(&rf).Update("refund_at", now)
			s.writeOrderLog(rf.OrderID, "refund_approved", 0, enum.RefundStatusApproved, "退款审核通过", op)
		} else {
			s.writeOrderLog(rf.OrderID, "refund_rejected", 0, enum.RefundStatusRejected, "退款审核驳回: "+remark, op)
		}

		return nil
	})
}

func (s *Service) ListRefunds(op Operator, orderID int64) ([]model.OrderRefund, error) {
	return s.OrderRepo.ListRefunds(op.CompanyID, orderID)
}
