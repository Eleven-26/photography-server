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

func (s *Service) CreatePayment(ctx context.Context, op Operator, orderID int64, req dto.PaymentCreateReq) (*model.OrderPayment, error) {
	o, err := s.OrderRepo.GetByID(ctx, op.CompanyID, orderID)
	if err != nil {
		return nil, errs.NotFound(errs.ErrOrderNotFound)
	}
	if o.Status == enum.OrderStatusCompleted || o.Status == enum.OrderStatusCancelled {
		return nil, errs.BadRequest(errs.ErrOrderCompleted)
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

	return repository.Tx(func(tx *gorm.DB) error {
		now := time.Now().Format("2006-01-02 15:04:05")

		// 1. 事务内锁定读取订单（基于锁定前快照做金额与状态判断，防并发重复确认）
		o, err := s.OrderRepo.WithTx(tx).GetByIDForUpdate(ctx, op.CompanyID, p.OrderID)
		if err != nil {
			return errs.NotFound(errs.ErrOrderNotFound)
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
		if err := s.OrderRepo.WithTx(tx).Update(ctx, op.CompanyID, p.OrderID, map[string]interface{}{
			"paid_amt":       gorm.Expr("paid_amt + ?", p.Amount),
			"payment_status": enum.PaymentStatusConfirmed,
		}); err != nil {
			return err
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
			if o.PaidAmt+p.Amount >= o.TotalAmt {
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
}

func (s *Service) ListPayments(ctx context.Context, op Operator, orderID int64) ([]model.OrderPayment, error) {
	return s.OrderRepo.ListPayments(ctx, op.CompanyID, orderID)
}

func (s *Service) GetUnconfirmedPayments(ctx context.Context, op Operator, page, pageSize int) ([]model.OrderPayment, int64, error) {
	return s.OrderRepo.GetUnconfirmedPayments(ctx, op.CompanyID, page, pageSize)
}

func (s *Service) GetTodayStats(ctx context.Context, op Operator) (confirmed float64, pending float64, err error) {
	return s.OrderRepo.GetTodayStats(ctx, op.CompanyID)
}
