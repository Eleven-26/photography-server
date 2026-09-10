package service

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"photography-server/internal/domain"
	"photography-server/internal/enum"
	"photography-server/internal/model"
	"photography-server/internal/pkg/errs"
	"photography-server/internal/presentation/dto"
	"photography-server/internal/repository"
)

// paymentTypeSet 收款类型白名单
var paymentTypeSet = map[string]bool{"deposit": true, "final": true, "addon": true}

// CreatePayment 录入收款记录。
// 约束（#8）：
//  1. amount 必须 > 0（负数/零直接拒绝，binding:"required" 对 float64 只保证非零不够）；
//  2. Type 必须为 deposit/final/addon 白名单；
//  3. 申请金额不得超过订单剩余应收（total_amt - paid_amt），防止超额收款；
//  4. 累计校验在确认收款事务内加行锁再做最终判定（见 ConfirmPayment）。
func (s *Service) CreatePayment(ctx context.Context, op Operator, orderID int64, req dto.PaymentCreateReq) (*model.OrderPayment, error) {
	if req.Amount <= 0 {
		return nil, errs.BadRequest("收款金额必须大于 0")
	}
	if !paymentTypeSet[req.Type] {
		return nil, errs.BadRequest("收款类型不合法（deposit/final/addon）")
	}
	o, err := s.OrderRepo.GetByID(ctx, op.CompanyID, orderID)
	if err != nil {
		return nil, errs.NotFound(errs.ErrOrderNotFound)
	}
	if o.Status == enum.OrderStatusCompleted || o.Status == enum.OrderStatusCancelled {
		return nil, errs.BadRequest(errs.ErrOrderCompleted)
	}
	// 金额上限：已收 + 本次 <= 订单总额（留 0.5 分舍入余量）
	if o.PaidAmt+req.Amount > o.TotalAmt+domain.FenEps() {
		return nil, errs.BadRequest("收款金额超过订单剩余应收")
	}

	p := model.OrderPayment{
		TenantBase: model.TenantBase{
			Base:      model.Base{CreatedBy: op.UserID, UpdatedBy: op.UserID},
			CompanyID: op.CompanyID,
		},
		OrderID:    orderID,
		Code:       domain.GenCode("PM"),
		CustomerID: o.CustomerID,
		Type:       req.Type,
		Amount:     req.Amount,
		MethodID:   req.MethodID,
		PaidAt:     strPtr(req.PaidAt),
		Voucher:    req.Voucher,
		Remark:     req.Remark,
		Status:     enum.PaymentStatusPending,
	}

	if err := s.OrderRepo.CreatePayment(ctx, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

func (s *Service) ConfirmPayment(ctx context.Context, op Operator, id int64) error {
	p, err := s.OrderRepo.GetPaymentByID(ctx, op.CompanyID, id)
	if err != nil {
		return errs.NotFound(errs.ErrPaymentNotFound)
	}
	if p.Status == enum.PaymentStatusConfirmed {
		return errs.BadRequest(errs.ErrPaymentConfirmed)
	}

	err = repository.Tx(func(tx *gorm.DB) error {
		now := time.Now().Format("2006-01-02 15:04:05")

		// 1. 事务内锁定读取订单（基于锁定前快照做金额与状态判断，防并发重复确认/超额收款）
		o, err := s.OrderRepo.WithTx(tx).GetByIDForUpdate(ctx, op.CompanyID, p.OrderID)
		if err != nil {
			return errs.NotFound(errs.ErrOrderNotFound)
		}
		// 调度费（type=reschedule，客户改期产生的费用）独立于订单套餐应收：
		// 不占订单额度、不计入订单已收，否则「已收」会超过「订单总额」，套餐应收/剩余应收口径被污染。
		isRescheduleFee := p.Type == rescheduleFeePaymentType

		// 1.1 收款确认时按最新订单快照二次校验：已收 + 本次不得超额（录入后订单可能被退款/改价）
		if !isRescheduleFee && o.PaidAmt+p.Amount > o.TotalAmt+domain.FenEps() {
			return errs.BadRequest("确认后收款将超过订单剩余应收，请核对金额")
		}

		// 2. 收款记录置为已确认（CAS：仅当仍为待核验时生效，防并发重复确认）
		ok, err := s.OrderRepo.WithTx(tx).ConfirmPaymentPending(ctx, op.CompanyID, id, map[string]interface{}{
			"status":        enum.PaymentStatusConfirmed,
			"operator_id":   op.UserID,
			"operator_name": op.Username,
		})
		if err != nil {
			return err
		}
		if !ok {
			return errs.BadRequest(errs.ErrPaymentConfirmed)
		}

		// 3. 累加订单已收金额（带租户过滤，同一事务连接）
		// 调度费不计入订单已收、也不改 payment_status —— 它不在订单套餐应收范围内。
		newPaid := o.PaidAmt + p.Amount
		if !isRescheduleFee {
			updates := map[string]interface{}{
				"paid_amt": gorm.Expr("paid_amt + ?", p.Amount),
			}
			// 3.1 payment_status 由金额推导：全额收齐才标"已确认"，部分收款保持原状态
			if st, ok := domain.DerivePaymentStatus(newPaid, o.RefundAmt, o.TotalAmt); ok {
				updates["payment_status"] = st
			}
			if err := s.OrderRepo.WithTx(tx).Update(ctx, op.CompanyID, p.OrderID, updates); err != nil {
				return err
			}
		}

		// 4. 状态流转：定金支付后进入待拍摄；尾款结清后订单完成
		if p.Type == "deposit" && o.Status == enum.OrderStatusPendingDeposit {
			if err := s.OrderRepo.WithTx(tx).Update(ctx, op.CompanyID, p.OrderID, map[string]interface{}{"status": enum.OrderStatusPendingShoot}); err != nil {
				return err
			}
			if err := s.writeOrderLogTx(ctx, tx, p.OrderID, "pay_deposit", 0, enum.OrderStatusPendingShoot, "定金支付确认", op); err != nil {
				return err
			}
		}
		if (p.Type == "final" || p.Type == "addon") && o.Status == enum.OrderStatusPendingDelivery {
			if newPaid >= o.TotalAmt-domain.FenEps() {
				if err := s.OrderRepo.WithTx(tx).Update(ctx, op.CompanyID, p.OrderID, map[string]interface{}{"status": enum.OrderStatusCompleted, "finished_at": now}); err != nil {
					return err
				}
				if err := s.writeOrderLogTx(ctx, tx, p.OrderID, "pay_final", enum.OrderStatusPendingDelivery, enum.OrderStatusCompleted, "尾款支付确认，订单完成", op); err != nil {
					return err
				}
			}
		}

		return nil
	})
	if err != nil {
		return err
	}
	// 5. 事务提交后再通知客户：到账确认是客户最关心的资金节点（通知失败不回滚已生效的核验）
	if p.Type == rescheduleFeePaymentType {
		s.NotifyClient(ctx, op, p.CustomerID, "finance", "改期调度费已确认",
			fmt.Sprintf("调度费 %.2f 元已核验到账", p.Amount), "payment", p.ID)
	} else {
		s.NotifyClient(ctx, op, p.CustomerID, "finance", "收款已确认到账",
			fmt.Sprintf("已确认到账 %.2f 元", p.Amount), "payment", p.ID)
	}
	return nil
}

func (s *Service) ListPayments(ctx context.Context, op Operator, orderID int64) ([]model.OrderPayment, error) {
	return s.OrderRepo.ListPayments(ctx, op.CompanyID, orderID)
}

// DeletePayment 删除未确认的收款记录。
// 已确认收款已计入订单已收金额，直接删除会造成"钱收了但账面没记录"，
// 因此只允许删除待核验/待支付状态；已确认的冲销必须走退款流程（refund/apply）。
func (s *Service) DeletePayment(ctx context.Context, op Operator, id int64) error {
	p, err := s.OrderRepo.GetPaymentByID(ctx, op.CompanyID, id)
	if err != nil {
		return errs.NotFound(errs.ErrPaymentNotFound)
	}
	if p.Status == enum.PaymentStatusConfirmed || p.Status == enum.PaymentStatusRefunded {
		return errs.BadRequest("已确认的收款不可删除，请走退款流程")
	}
	return repository.Tx(func(tx *gorm.DB) error {
		if err := s.OrderRepo.WithTx(tx).DeletePayment(ctx, op.CompanyID, id); err != nil {
			return err
		}
		return s.writeOrderLogTx(ctx, tx, p.OrderID, "delete_payment", 0, 0,
			fmt.Sprintf("删除未确认收款 %s（¥%.2f）", p.Code, p.Amount), op)
	})
}

func (s *Service) GetUnconfirmedPayments(ctx context.Context, op Operator, page, pageSize int) ([]model.OrderPayment, int64, error) {
	return s.OrderRepo.GetUnconfirmedPayments(ctx, op.CompanyID, page, pageSize)
}

func (s *Service) GetTodayStats(ctx context.Context, op Operator) (confirmed float64, pending float64, err error) {
	return s.OrderRepo.GetTodayStats(ctx, op.CompanyID)
}
