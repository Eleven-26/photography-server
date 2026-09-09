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

// CreateRefund 管理端发起退款申请。
// 约束（#7）：
//  1. 退款申请金额不得超过「已收 - 已退」（剩余可退），防止超额退/重复退；
//  2. 同一订单同时最多一张申请中的退款单；
//  3. 申请与金额校验放入事务 + 行锁，避免并发创建多张申请单导致累计超额。
func (s *Service) CreateRefund(ctx context.Context, op Operator, orderID int64, req dto.RefundCreateReq) (*model.OrderRefund, error) {
	var rf *model.OrderRefund
	err := repository.Tx(func(tx *gorm.DB) error {
		o, err := s.OrderRepo.WithTx(tx).GetByIDForUpdate(ctx, op.CompanyID, orderID)
		if err != nil {
			return errs.NotFound(errs.ErrOrderNotFound)
		}
		if o.Status == enum.OrderStatusCancelled {
			return errs.BadRequest(errs.ErrRefundCancelled)
		}

		// 剩余可退 = 已收 - 已退；已退完/未收则不可再退
		refundable := o.PaidAmt - o.RefundAmt
		if refundable <= 0 {
			return errs.BadRequest(errs.ErrRefundZero)
		}

		amount := req.Amount
		if amount == 0 {
			amount = o.PaidAmt // 未指定金额默认按全部已收申请（再经比例折算）
		}
		if amount <= 0 {
			return errs.BadRequest(errs.ErrRefundZero)
		}
		// 申请基数不得超过剩余可退（全额比例 1.0 时也不至于超额）
		if amount > refundable+domain.FenEps() {
			return errs.BadRequest("退款金额超过订单剩余可退金额")
		}

		// 同一订单只能有一张申请中的退款单（防多笔小额申请拆分审核绕过累计上限）
		applying, err := s.OrderRepo.WithTx(tx).ListRefunds(ctx, op.CompanyID, orderID)
		if err != nil {
			return err
		}
		for _, a := range applying {
			if a.Status == enum.RefundStatusApplying {
				return errs.BadRequest("该订单已有申请中的退款单，请先处理")
			}
		}

		shootTime, _ := domain.ParseShootDate(dateOrEmpty(o.ShootDate))
		hoursBeforeShoot := time.Until(shootTime).Hours()
		ratio, rule := domain.RefundRatio(time.Duration(hoursBeforeShoot) * time.Hour)
		refundAmt := domain.Round2(amount * ratio)
		if refundAmt <= 0 {
			return errs.BadRequest(errs.ErrRefundNoTime)
		}
		if refundAmt > refundable+domain.FenEps() {
			return errs.BadRequest("退款金额超过订单剩余可退金额")
		}

		rf = &model.OrderRefund{
			TenantBase: model.TenantBase{
				Base:      model.Base{CreatedBy: op.UserID, UpdatedBy: op.UserID},
				CompanyID: op.CompanyID,
			},
			OrderID:    orderID,
			Code:       domain.GenCode("RF"),
			CustomerID: o.CustomerID,
			Amount:     refundAmt,
			Reason:     req.Reason,
			RefundRule: rule,
			Status:     enum.RefundStatusApplying,
			ApplyBy:    op.UserID,
			ApplyName:  op.Username,
		}
		return s.OrderRepo.WithTx(tx).CreateRefund(ctx, rf)
	})
	if err != nil {
		return nil, err
	}
	return rf, nil
}

func (s *Service) AuditRefund(ctx context.Context, op Operator, id int64, approved bool, remark string) error {
	rf, err := s.OrderRepo.GetRefundByID(ctx, op.CompanyID, id)
	if err != nil {
		return errs.NotFound(errs.ErrRefundNotFound)
	}
	if rf.Status != enum.RefundStatusApplying {
		return errs.BadRequest(errs.ErrRefundProcessed)
	}

	now := time.Now().Format("2006-01-02 15:04:05")
	status := enum.RefundStatusRejected
	if approved {
		status = enum.RefundStatusApproved
	}

	return repository.Tx(func(tx *gorm.DB) error {
		// 1. 更新退款单状态（CAS：仅当仍为"申请中"时生效，防并发重复审核）
		ok, err := s.OrderRepo.WithTx(tx).AuditRefundApplying(ctx, op.CompanyID, id, map[string]interface{}{
			"status":       status,
			"audit_by":     op.UserID,
			"audit_at":     now,
			"audit_remark": remark,
		})
		if err != nil {
			return err
		}
		if !ok {
			return errs.BadRequest(errs.ErrRefundProcessed)
		}

		if approved {
			// 2. 事务内锁定读取订单，避免并发审核导致退款金额重复累加
			o, err := s.OrderRepo.WithTx(tx).GetByIDForUpdate(ctx, op.CompanyID, rf.OrderID)
			if err != nil {
				return errs.NotFound(errs.ErrOrderNotFound)
			}
			// 2.1 审核时二次校验：本次退款不得让累计退款超过已收（部分审核也受此约束）
			if o.PaidAmt-o.RefundAmt < rf.Amount-domain.FenEps() {
				return errs.BadRequest("退款金额超过订单剩余可退金额，无法通过")
			}
			// 3. 累加订单已退金额（带租户过滤，同一事务连接）
			newRefund := o.RefundAmt + rf.Amount
			updates := map[string]interface{}{
				"refund_amt": gorm.Expr("refund_amt + ?", rf.Amount),
			}
			// 4. payment_status 由金额推导：全额退清才标"已退款"，部分退款保持原状态
			if st, ok := domain.DerivePaymentStatus(o.PaidAmt, newRefund, o.TotalAmt); ok {
				updates["payment_status"] = st
			}
			if err := s.OrderRepo.WithTx(tx).Update(ctx, op.CompanyID, rf.OrderID, updates); err != nil {
				return err
			}
			if err := s.OrderRepo.WithTx(tx).UpdateRefund(ctx, op.CompanyID, id, map[string]interface{}{"refund_at": now}); err != nil {
				return err
			}
			if err := s.writeOrderLogTx(ctx, tx, rf.OrderID, "refund_approved", 0, enum.RefundStatusApproved, "退款审核通过", op); err != nil {
				return err
			}
		} else {
			if err := s.writeOrderLogTx(ctx, tx, rf.OrderID, "refund_rejected", 0, enum.RefundStatusRejected, "退款审核驳回: "+remark, op); err != nil {
				return err
			}
		}

		return nil
	})
}

func (s *Service) ListRefunds(ctx context.Context, op Operator, orderID int64) ([]model.OrderRefund, error) {
	return s.OrderRepo.ListRefunds(ctx, op.CompanyID, orderID)
}

// dateOrEmpty 从 *string 取日期字符串
func dateOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
