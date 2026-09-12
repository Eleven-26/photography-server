package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"photography-server/internal/domain"
	"photography-server/internal/enum"
	"photography-server/internal/model"
	"photography-server/internal/pkg/errs"
	"photography-server/internal/pkg/logger"
	"photography-server/internal/presentation/dto"
)

// client_finance 客户端（H5/小程序）资金与售后链路：
// 退款进度与到账确认、改期单列表、收款记录与「我已完成转账」登记、收款方式读取、拍前准备已读。
//
// 与 PC 端的差别**只在归属口径**：
//   - PC 用 Operator（租户内任意订单可操作），客户只能用挂在自己名下的订单；
//   - PC 的登记（/payment/create）与核验（/payment/confirm）由同侧员工完成，
//     客户侧只做「登记」（status=1 待核验），核验仍在员工端，资金确认权不下放。
//
// 复用同一批仓储与同一张表，避免"客户端另起一套收款表"造成对账分裂。

// clientOwnedOrder 取订单并校验归属当前客户（客户端售后链路公共前置）。
// 「不存在」与「非本人」返回同一 403/404 口径，避免用 ID 遍历探测他人订单。
func (s *Service) clientOwnedOrder(ctx context.Context, cu *ClientUser, orderID int64) (*model.Order, error) {
	o, err := s.OrderRepo.GetByID(ctx, cu.CompanyID, orderID)
	if err != nil {
		return nil, errs.NotFound(errs.ErrOrderNotFound)
	}
	if err := clientOrderOwned(o, cu); err != nil {
		return nil, err
	}
	return o, nil
}

// ---------------------------------------------------------------------
// 退款进度（H5 画板 C21）
// ---------------------------------------------------------------------

// ClientRefunds 我的订单退款记录（倒序，最新在前）。
// 与 ClientOrderDetail 的 refunds 同源同表，此处为独立页面提供的窄接口。
func (s *Service) ClientRefunds(ctx context.Context, cu *ClientUser, orderID int64) ([]model.OrderRefund, error) {
	if _, err := s.clientOwnedOrder(ctx, cu, orderID); err != nil {
		return nil, err
	}
	return s.OrderRepo.ListRefunds(ctx, cu.CompanyID, orderID)
}

// ClientConfirmRefundReceived 客户确认收到退款（写 customer_confirm_at）。
// 与 PC 端审批闭环：P 侧「审核通过 + 退款」→ C 侧「我收到了」，两边都签字才算走完。
// 幂等：已确认过直接返回成功（客户可能重复点）。
func (s *Service) ClientConfirmRefundReceived(ctx context.Context, cu *ClientUser, refundID int64) error {
	rf, err := s.OrderRepo.GetRefundByID(ctx, cu.CompanyID, refundID)
	if err != nil {
		return errs.NotFound(errs.ErrRefundNotFound)
	}
	// 归属：优先退款单上的 customer_id；早期数据未回填时回落到订单归属
	owned := rf.CustomerID > 0 && rf.CustomerID == cu.CustomerID
	if !owned {
		if o, err := s.OrderRepo.GetByID(ctx, cu.CompanyID, rf.OrderID); err == nil && o.CustomerID == cu.CustomerID {
			owned = true
		}
	}
	if !owned {
		return errs.NotFound(errs.ErrRefundNotFound)
	}
	if rf.CustomerConfirmAt != nil && *rf.CustomerConfirmAt != "" {
		return nil // 已确认：幂等
	}
	// 只有"已通过 / 已退款"才可确认收到——申请中/已驳回时确认没有意义，
	// 且会让 PC 侧误以为款项已结清。
	if rf.Status != enum.RefundStatusApproved && rf.Status != enum.RefundStatusDone {
		return errs.BadRequest("退款尚未通过或已驳回，暂不可确认收款")
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	return s.OrderRepo.UpdateRefund(ctx, cu.CompanyID, refundID, map[string]interface{}{
		"customer_confirm_at": now,
	})
}

// ---------------------------------------------------------------------
// 改期单列表（H5 改期进度页）
// ---------------------------------------------------------------------

// ClientReschedules 我的订单改期单列表。
// 与 ClientOrderDetail 的 reschedules 同源；独立接口供改期进度页直接拉取。
func (s *Service) ClientReschedules(ctx context.Context, cu *ClientUser, orderID int64) ([]model.OrderReschedule, error) {
	if _, err := s.clientOwnedOrder(ctx, cu, orderID); err != nil {
		return nil, err
	}
	return s.RescheduleRepo.ListByOrder(ctx, cu.CompanyID, orderID)
}

// ---------------------------------------------------------------------
// 收款记录 / 登记转账 / 收款方式（H5 支付页）
// ---------------------------------------------------------------------

// ClientPayments 我的订单收款记录（含待核验/已确认，供支付页展示登记状态）。
func (s *Service) ClientPayments(ctx context.Context, cu *ClientUser, orderID int64) ([]model.OrderPayment, error) {
	if _, err := s.clientOwnedOrder(ctx, cu, orderID); err != nil {
		return nil, err
	}
	return s.OrderRepo.ListPayments(ctx, cu.CompanyID, orderID)
}

// ClientRegisterPayment 客户登记转账（「我已完成转账，通知摄影师」）。
//
// 关键约束：
//  1. 资金不经平台（需求文档 §1）——本接口**只登记**，不确认到账：
//     落库 status=1（待核验），订单 paid_amt 不变，核验仍在员工端 /payment/confirm/:id。
//  2. 金额默认按「订单剩余应收」（total - paid）取，客户不传也无需自己算；
//     显式传值时不得让累计已收超过订单总额（与 PC CreatePayment 同一上限口径）。
//  3. 银行转账类收款方式必须带凭证，否则员工端无据可核。
func (s *Service) ClientRegisterPayment(ctx context.Context, cu *ClientUser, orderID int64, req dto.ClientPaymentMarkReq) (*model.OrderPayment, error) {
	o, err := s.clientOwnedOrder(ctx, cu, orderID)
	if err != nil {
		return nil, err
	}
	if !paymentTypeSet[req.Type] {
		return nil, errs.BadRequest("收款类型不合法（deposit/final/addon）")
	}
	if o.Status == enum.OrderStatusCompleted || o.Status == enum.OrderStatusCancelled {
		return nil, errs.BadRequest(errs.ErrOrderCompleted)
	}

	amount := domain.Round2(req.Amount)
	if amount <= 0 {
		amount = domain.Round2(o.TotalAmt - o.PaidAmt) // 不传 = 按剩余应收
	}
	if amount <= 0 {
		return nil, errs.BadRequest("订单已无待收金额，无需登记")
	}
	if o.PaidAmt+amount > o.TotalAmt+domain.FenEps() {
		return nil, errs.BadRequest("登记金额超过订单剩余应收，请核对后重试")
	}

	// 收款方式快照 + 渠道校验（方式可为空，表示客户自行线下转账）
	methodName := ""
	if req.MethodID > 0 {
		m, err := s.SettingsRepo.GetPaymentMethodByID(ctx, cu.CompanyID, req.MethodID)
		if err != nil {
			return nil, errs.BadRequest(errs.ErrPaymentMethodNotFound)
		}
		if m.Status != 1 {
			return nil, errs.BadRequest("该收款方式已停用，请选择其他方式")
		}
		methodName = m.Name
		if m.Type == "bank" && strings.TrimSpace(req.Voucher) == "" {
			return nil, errs.BadRequest("银行转账请上传转账凭证")
		}
	}

	now := time.Now()
	p := model.OrderPayment{
		TenantBase: model.TenantBase{
			Base: model.Base{
				CreatedAt: now, UpdatedAt: now,
				CreatedBy: cu.CustomerID, UpdatedBy: cu.CustomerID,
			},
			CompanyID: cu.CompanyID,
		},
		OrderID:    orderID,
		Code:       domain.GenCode("PM"),
		CustomerID: cu.CustomerID,
		Type:       req.Type,
		Amount:     amount,
		MethodID:   req.MethodID,
		MethodName: methodName,
		Status:     enum.PaymentStatusPending, // 待核验：到账确认权在员工端
		PaidAt:     strPtr(req.PaidAt),
		Voucher:    req.Voucher,
		Remark:     req.Remark,
	}
	if err := s.OrderRepo.CreatePayment(ctx, &p); err != nil {
		return nil, err
	}

	// 留痕 + 通知负责人：登记本身不改变资金状态，但要让员工知道"该核对了"
	op := clientOperator(cu)
	if err := s.writeOrderLog(ctx, orderID, "client_pay_mark", o.Status, o.Status,
		fmt.Sprintf("客户登记%s %.2f 元（待核验）", paymentTypeName(req.Type), amount), op); err != nil {
		logger.Warnf("ClientRegisterPayment: writeOrderLog failed, orderID=%d, err=%v", orderID, err)
	}
	s.NotifyStaff(ctx, op, o.OwnerID, "finance", "客户已登记付款",
		fmt.Sprintf("客户为订单 %s 登记%s ¥%.2f，请核对到账后确认", o.Code, paymentTypeName(req.Type), amount),
		"payment", p.ID)
	return &p, nil
}

// ClientPaymentMethods 客户可见的收款方式（只出启用项）。
//
// 为什么不复用 PC 的 /settings/payment-method/list：那是管理端配置接口（settings:view），
// 且返回内部 status/sort 等字段；客户侧只需要"照着付款"的最小信息集。
func (s *Service) ClientPaymentMethods(ctx context.Context, cu *ClientUser) ([]dto.ClientPaymentMethodResp, error) {
	list, err := s.SettingsRepo.ListPaymentMethods(ctx, cu.CompanyID)
	if err != nil {
		return nil, err
	}
	out := make([]dto.ClientPaymentMethodResp, 0, len(list))
	for _, m := range list {
		if m.Status != 1 { // 停用项不下发
			continue
		}
		out = append(out, dto.ClientPaymentMethodResp{
			ID:          m.ID,
			Name:        m.Name,
			Type:        m.Type,
			AccountName: m.AccountName,
			AccountNo:   m.AccountNo,
			Qrcode:      m.Qrcode,
		})
	}
	return out, nil
}

// ---------------------------------------------------------------------
// 拍前准备已读（H5 订单详情）
// ---------------------------------------------------------------------

// ClientReadOrderPrep 客户确认已读「拍前准备清单」（写 biz_order.prep_read_at）。
// 幂等：已读不覆盖首次阅读时间——工作室关心的是"客户首次看到"的时间点。
func (s *Service) ClientReadOrderPrep(ctx context.Context, cu *ClientUser, orderID int64) error {
	o, err := s.clientOwnedOrder(ctx, cu, orderID)
	if err != nil {
		return err
	}
	if o.PrepReadAt != nil && *o.PrepReadAt != "" {
		return nil
	}
	if strings.TrimSpace(o.PrepContent) == "" {
		return errs.BadRequest("该订单暂无拍前准备内容")
	}
	return s.OrderRepo.Update(ctx, cu.CompanyID, orderID, map[string]interface{}{
		"prep_read_at": time.Now().Format("2006-01-02 15:04:05"),
	})
}
