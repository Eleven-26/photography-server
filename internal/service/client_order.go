package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"photography-server/internal/domain"
	"photography-server/internal/enum"
	"photography-server/internal/model"
	"photography-server/internal/pkg/errs"
	"photography-server/internal/pkg/logger"
	"photography-server/internal/presentation/dto"
	"photography-server/internal/repository"
)

// client_order 客户端（H5/小程序）订单后续操作：
// 改期申请、退款申请、评价、选片、确认加片、确认成片。全部带客户归属校验。

// ClientRescheduleApply 客户发起改期申请（同一订单同时最多一张待确认改期单）。
// 费用按工作室改期政策计算：>=FreeHours 免费、区间收调度费、<MinHours 拒绝。
func (s *Service) ClientRescheduleApply(ctx context.Context, cu *ClientUser, orderID int64, req dto.ClientRescheduleReq) (*model.OrderReschedule, error) {
	o, err := s.OrderRepo.GetByID(ctx, cu.CompanyID, orderID)
	if err != nil {
		return nil, errs.NotFound(errs.ErrOrderNotFound)
	}
	if err := clientOrderOwned(o, cu); err != nil {
		return nil, err
	}
	switch o.Status {
	case enum.OrderStatusPendingConfirm, enum.OrderStatusPendingDeposit, enum.OrderStatusPendingShoot:
		// 可改期状态
	default:
		return nil, errs.BadRequest("当前订单状态不可改期")
	}
	if req.NewDate == "" || req.NewTime == "" {
		return nil, errs.BadRequest("请选择新的拍摄日期与时段")
	}
	// #20：仅"确实没有待确认改期单"（ErrRecordNotFound）才放行；
	// 查询出错（DB 抖动）直接返回，避免约束在故障期失效导致重复改期单
	if _, err := s.RescheduleRepo.GetPendingByOrder(ctx, cu.CompanyID, orderID); err == nil {
		return nil, errs.BadRequest("已有待确认的改期申请，请耐心等待")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	// 距原拍摄开始的小时数 → 费用档位
	origDate := ""
	if o.ShootDate != nil {
		origDate = *o.ShootDate
	}
	origStart, _ := time.ParseInLocation("2006-01-02 15:04", origDate+" 00:00", time.Local)
	if hourPart(o.ShootTime) != "" {
		if t, err := time.ParseInLocation("2006-01-02 15:04", origDate+" "+hourPart(o.ShootTime), time.Local); err == nil {
			origStart = t
		}
	}
	feeType, feeAmt, _ := domain.RescheduleFee(time.Until(origStart), o.TotalAmt, s.clientReschedulePolicy(ctx, cu.CompanyID))
	if feeType == enum.RescheduleFeeForbidden {
		return nil, errs.BadRequest("距拍摄不足24小时，不可改期，请联系工作室")
	}

	now := time.Now()
	rs := model.OrderReschedule{
		TenantBase:   model.TenantBase{Base: model.Base{CreatedAt: now, UpdatedAt: now}, CompanyID: cu.CompanyID},
		Code:         domain.GenCode("RS"),
		OrderID:      orderID,
		CustomerID:   cu.CustomerID,
		OriginalDate: origDate,
		OriginalTime: o.ShootTime,
		NewDate:      req.NewDate,
		NewTime:      req.NewTime,
		FeeType:      feeType,
		FeeAmount:    feeAmt,
		ReasonLabel:  req.ReasonLabel,
		Reason:       req.Reason,
		Status:       enum.RescheduleStatusPending,
		ApplySource:  2, // 客户申请
	}
	if err := s.RescheduleRepo.Create(ctx, &rs); err != nil {
		return nil, err
	}
	if err := s.writeOrderLog(ctx, orderID, "client_reschedule", o.Status, o.Status,
		"客户申请改期至 "+req.NewDate+" "+req.NewTime, clientOperator(cu)); err != nil {
		logger.Warnf("ClientRescheduleApply: writeOrderLog failed, orderID=%d, err=%v", orderID, err)
	}
	// 站内通知：改期申请待审批（失败不阻断申请）
	s.NotifyStaff(ctx, clientOperator(cu), o.OwnerID, "order", "改期申请待审批",
		fmt.Sprintf("%s 申请将订单 %s 改期至 %s %s，请尽快处理", cu.Name, o.Code, req.NewDate, req.NewTime),
		"order", orderID)
	return &rs, nil
}

// hourPart 从 "09:00-11:00" 中取开始小时部分
func hourPart(timeRange string) string {
	for i := 0; i < len(timeRange); i++ {
		if timeRange[i] == '-' {
			return timeRange[:i]
		}
	}
	return timeRange
}

// ClientRescheduleCancel 客户撤回待确认的改期申请
func (s *Service) ClientRescheduleCancel(ctx context.Context, cu *ClientUser, rescheduleID int64) error {
	rs, err := s.RescheduleRepo.GetByID(ctx, cu.CompanyID, rescheduleID)
	if err != nil {
		return errs.NotFound("改期单不存在")
	}
	if rs.CustomerID != cu.CustomerID {
		return errs.Forbidden("无权操作该改期单")
	}
	if rs.Status != enum.RescheduleStatusPending {
		return errs.BadRequest("该改期单已处理，不可撤回")
	}
	return s.RescheduleRepo.Update(ctx, cu.CompanyID, rescheduleID, map[string]interface{}{
		"status": enum.RescheduleStatusCancelled,
	})
}

// ClientRefundApply 客户申请退款（apply_source=2），金额按退款规则试算
func (s *Service) ClientRefundApply(ctx context.Context, cu *ClientUser, orderID int64, req dto.ClientRefundReq) (*model.OrderRefund, error) {
	o, err := s.OrderRepo.GetByID(ctx, cu.CompanyID, orderID)
	if err != nil {
		return nil, errs.NotFound(errs.ErrOrderNotFound)
	}
	if err := clientOrderOwned(o, cu); err != nil {
		return nil, err
	}
	if o.Status == enum.OrderStatusCancelled {
		return nil, errs.BadRequest(errs.ErrRefundCancelled)
	}
	if o.PaidAmt <= 0 {
		return nil, errs.BadRequest(errs.ErrRefundNoTime)
	}

	// 按距拍摄时间试算可退金额（#16：统一 domain.ParseShootDate，本地时区解析）
	shootDate := ""
	if o.ShootDate != nil {
		shootDate = *o.ShootDate
	}
	shootTime, _ := domain.ParseShootDate(shootDate)
	hours := time.Until(shootTime).Hours()
	preview := domain.CalcRefundPreview(time.Duration(hours) * time.Hour)
	refundAmt := domain.Round2(o.PaidAmt * preview.Ratio)
	if refundAmt <= 0 {
		return nil, errs.BadRequest(errs.ErrRefundNoTime)
	}

	now := time.Now()
	rf := model.OrderRefund{
		TenantBase:  model.TenantBase{Base: model.Base{CreatedAt: now, UpdatedAt: now}, CompanyID: cu.CompanyID},
		OrderID:     orderID,
		Code:        domain.GenCode("RF"),
		CustomerID:  cu.CustomerID,
		Amount:      refundAmt,
		Reason:      req.Reason,
		ReasonLabel: req.ReasonLabel,
		ApplySource: 2, // 客户H5/小程序
		RefundRule:  preview.Rule,
		Status:      enum.RefundStatusApplying,
		ApplyBy:     cu.CustomerID,
		ApplyName:   cu.Name,
	}
	if err := s.OrderRepo.CreateRefund(ctx, &rf); err != nil {
		return nil, err
	}
	if err := s.writeOrderLog(ctx, orderID, "client_refund", o.Status, o.Status, "客户申请退款", clientOperator(cu)); err != nil {
		logger.Warnf("ClientRefundApply: writeOrderLog failed, orderID=%d, err=%v", orderID, err)
	}
	// 站内通知：退款申请待审批（失败不阻断申请）
	s.NotifyStaff(ctx, clientOperator(cu), o.OwnerID, "finance", "退款申请待审批",
		fmt.Sprintf("%s 申请退款 ¥%.2f（订单 %s），请尽快处理", cu.Name, rf.Amount, o.Code),
		"refund", orderID)
	return &rf, nil
}

// ClientReviewCreate 订单评价（已完成订单、每单一评）
func (s *Service) ClientReviewCreate(ctx context.Context, cu *ClientUser, orderID int64, req dto.ClientReviewReq) (*model.OrderReview, error) {
	o, err := s.OrderRepo.GetByID(ctx, cu.CompanyID, orderID)
	if err != nil {
		return nil, errs.NotFound(errs.ErrOrderNotFound)
	}
	if err := clientOrderOwned(o, cu); err != nil {
		return nil, err
	}
	if o.Status != enum.OrderStatusCompleted {
		return nil, errs.BadRequest("订单完成后方可评价")
	}
	if req.Rating < 1 || req.Rating > 5 {
		return nil, errs.BadRequest("评分需在 1-5 之间")
	}
	// #20：仅"确实未评价过"（ErrRecordNotFound）才放行，查询出错直接返回
	if _, err := s.ReviewRepo.GetByOrderID(ctx, cu.CompanyID, orderID); err == nil {
		return nil, errs.BadRequest("该订单已评价过")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	now := time.Now()
	name := cu.Name
	if req.IsAnonymous == 1 {
		name = "匿名用户"
	}
	rv := model.OrderReview{
		TenantBase:   model.TenantBase{Base: model.Base{CreatedAt: now, UpdatedAt: now}, CompanyID: cu.CompanyID},
		OrderID:      orderID,
		CustomerID:   cu.CustomerID,
		CustomerName: name,
		Rating:       req.Rating,
		Content:      req.Content,
		Images:       req.Images,
		IsAnonymous:  req.IsAnonymous,
	}
	if err := s.ReviewRepo.Create(ctx, &rv); err != nil {
		return nil, err
	}
	return &rv, nil
}

// ClientDeliveryDetail 客户查看交付单与明细（选片页/成片页）。
//
// 入参是 **order_id**（按订单反查交付单），与 PC 端 `/delivery/detail/:id` 的语义一致
// （service.ListDeliveryItemsByOrder 亦为 order_id）——此前客户端这条路由用的是
// delivery_id，与 PC 同 Path 不同语义，前端按 order_id 调用必然 404（2026-09-12 联调修正）。
// 交付单尚未创建时返回 (nil, 空列表)，前端展示空态而非报错。
func (s *Service) ClientDeliveryDetail(ctx context.Context, cu *ClientUser, orderID int64) (*model.Delivery, []model.DeliveryItem, error) {
	if _, err := s.clientOwnedOrder(ctx, cu, orderID); err != nil {
		return nil, nil, err
	}
	d, err := s.DeliveryRepo.GetByOrderID(ctx, cu.CompanyID, orderID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, []model.DeliveryItem{}, nil
		}
		return nil, nil, err
	}
	items, err := s.DeliveryRepo.ListItems(ctx, cu.CompanyID, d.ID)
	if err != nil {
		return nil, nil, err
	}
	return d, items, nil
}

// ClientSelectPhotos 客户选片：勾选样片；超出套餐精修张数部分计加片费。
// 规则：选片截止前可反复调整；截止后锁定（服务端兜底，前端倒计时）。
func (s *Service) ClientSelectPhotos(ctx context.Context, cu *ClientUser, deliveryID int64, itemIDs []int64) error {
	d, err := s.DeliveryRepo.GetByID(ctx, cu.CompanyID, deliveryID)
	if err != nil {
		return errs.NotFound(errs.ErrDeliveryNotFound)
	}
	o, err := s.OrderRepo.GetByID(ctx, cu.CompanyID, d.OrderID)
	if err != nil {
		return errs.NotFound(errs.ErrOrderNotFound)
	}
	if err := clientOrderOwned(o, cu); err != nil {
		return err
	}
	if d.Stage != enum.DeliveryStageSelecting {
		return errs.BadRequest(errs.ErrDeliveryStageInvalid)
	}
	// #23：加片费一旦确认已进入订单尾款，选片即锁定——防止"确认后改选变少"造成
	// 订单金额与选择结果脱钩（改选不冲销已入账金额）。如需调整请联系工作室处理。
	if d.ExtraConfirmed == 1 {
		return errs.BadRequest("已确认加片费用，选片已锁定；如需调整请联系工作室")
	}
	if d.SelectDeadline != nil && *d.SelectDeadline != "" {
		if deadline, err := time.ParseInLocation("2006-01-02 15:04:05", *d.SelectDeadline, time.Local); err == nil {
			if time.Now().After(deadline) {
				return errs.BadRequest("选片已截止，如需调整请联系工作室")
			}
		}
	}

	items, err := s.DeliveryRepo.ListItems(ctx, cu.CompanyID, deliveryID)
	if err != nil {
		return err
	}
	selSet := map[int64]bool{}
	for _, id := range itemIDs {
		selSet[id] = true
	}
	selected := 0
	for _, it := range items {
		want := 0
		if selSet[it.ID] {
			want = 1
			selected++
		}
		if it.IsSelected != want {
			if err := s.DeliveryRepo.UpdateItem(ctx, cu.CompanyID, it.ID, map[string]interface{}{"is_selected": want}); err != nil {
				return err
			}
		}
	}

	// 加片费：超出套餐精修张数 × 加片单价
	pkg, _ := s.PackageRepo.GetByID(ctx, cu.CompanyID, o.PackageID)
	included := 0
	unitPrice := 0.0
	if pkg != nil {
		included = pkg.PhotosIncluded
		unitPrice = pkg.AddonUnitPrice
	}
	extraCount, extraFee := domain.ExtraRetouchFee(selected, included, unitPrice)

	now := time.Now().Format("2006-01-02 15:04:05")
	return s.DeliveryRepo.Update(ctx, cu.CompanyID, deliveryID, map[string]interface{}{
		"selected_count":       selected,
		"extra_selected_count": extraCount,
		"extra_fee":            extraFee,
		"selected_at":          now,
	})
}

// ClientConfirmExtra 客户确认加片费用：加片金额计入尾款。
// 幂等（#22）：确认动作由事务内 CAS（UPDATE ... WHERE extra_confirmed=0）抢占，
// 并发双击/重复请求只有一个能真正累加金额；其余请求读到已确认状态后幂等返回成功。
func (s *Service) ClientConfirmExtra(ctx context.Context, cu *ClientUser, deliveryID int64) error {
	d, err := s.DeliveryRepo.GetByID(ctx, cu.CompanyID, deliveryID)
	if err != nil {
		return errs.NotFound(errs.ErrDeliveryNotFound)
	}
	o, err := s.OrderRepo.GetByID(ctx, cu.CompanyID, d.OrderID)
	if err != nil {
		return errs.NotFound(errs.ErrOrderNotFound)
	}
	if err := clientOrderOwned(o, cu); err != nil {
		return err
	}
	if d.ExtraConfirmed == 1 {
		return nil // 已确认：幂等成功
	}
	extraFee := d.ExtraFee
	return repository.Tx(func(tx *gorm.DB) error {
		// 1. CAS 抢占确认位：只有 extra_confirmed=0 → 1 的行受影响
		ok, err := s.DeliveryRepo.WithTx(tx).CasConfirmExtra(ctx, cu.CompanyID, deliveryID)
		if err != nil {
			return err
		}
		if !ok {
			// 并发下已被其他请求确认：重读确认状态，幂等返回
			d2, err := s.DeliveryRepo.WithTx(tx).GetByID(ctx, cu.CompanyID, deliveryID)
			if err != nil {
				return err
			}
			if d2.ExtraConfirmed == 1 {
				return nil
			}
			return errs.Conflict("加片确认状态异常，请刷新后重试")
		}
		// 2. 加片费进订单尾款（addon_amount 累加，final_amt/total_amt 同步）
		if err := s.OrderRepo.WithTx(tx).Update(ctx, cu.CompanyID, o.ID, map[string]interface{}{
			"addon_amount": gorm.Expr("addon_amount + ?", extraFee),
			"final_amt":    gorm.Expr("final_amt + ?", extraFee),
			"total_amt":    gorm.Expr("total_amt + ?", extraFee),
		}); err != nil {
			return err
		}
		return s.writeOrderLogTx(ctx, tx, o.ID, "client_confirm_extra", o.Status, o.Status,
			"客户确认加片，加收金额已计入尾款", clientOperator(cu))
	})
}

// ClientConfirmDelivery 客户确认成片：交付单 → 已交付，订单 → 已完成
func (s *Service) ClientConfirmDelivery(ctx context.Context, cu *ClientUser, deliveryID int64) error {
	d, err := s.DeliveryRepo.GetByID(ctx, cu.CompanyID, deliveryID)
	if err != nil {
		return errs.NotFound(errs.ErrDeliveryNotFound)
	}
	o, err := s.OrderRepo.GetByID(ctx, cu.CompanyID, d.OrderID)
	if err != nil {
		return errs.NotFound(errs.ErrOrderNotFound)
	}
	if err := clientOrderOwned(o, cu); err != nil {
		return err
	}
	if d.Stage != enum.DeliveryStagePendingConfirm {
		return errs.BadRequest("当前阶段不可确认成片")
	}
	now := time.Now()
	nowStr := now.Format("2006-01-02 15:04:05")
	return repository.Tx(func(tx *gorm.DB) error {
		if err := s.DeliveryRepo.WithTx(tx).Update(ctx, cu.CompanyID, deliveryID, map[string]interface{}{
			"stage":                 enum.DeliveryStageDelivered,
			"customer_confirmed_at": nowStr,
			"delivered_at":          nowStr,
		}); err != nil {
			return err
		}
		if !domain.OrderCanTransit(o.Status, enum.OrderStatusCompleted) {
			return errs.BadRequest(errs.ErrOrderStatusInvalid)
		}
		if err := s.OrderRepo.WithTx(tx).Update(ctx, cu.CompanyID, o.ID, map[string]interface{}{
			"status":      enum.OrderStatusCompleted,
			"finished_at": nowStr,
		}); err != nil {
			return err
		}
		return s.writeOrderLogTx(ctx, tx, o.ID, "client_confirm_delivery", o.Status, enum.OrderStatusCompleted,
			"客户确认成片，订单完成", clientOperator(cu))
	})
}

// ClientFeedbackSubmit 客户提交精修反馈（对某张精修成品提出修改意见）
func (s *Service) ClientFeedbackSubmit(ctx context.Context, cu *ClientUser, itemID int64, req dto.ClientFeedbackReq) error {
	var item model.DeliveryItem
	if err := s.DeliveryRepo.FirstItem(ctx, cu.CompanyID, itemID, &item); err != nil {
		return errs.NotFound("交付文件不存在")
	}
	d, err := s.DeliveryRepo.GetByID(ctx, cu.CompanyID, item.DeliveryID)
	if err != nil {
		return errs.NotFound(errs.ErrDeliveryNotFound)
	}
	o, err := s.OrderRepo.GetByID(ctx, cu.CompanyID, d.OrderID)
	if err != nil {
		return errs.NotFound(errs.ErrOrderNotFound)
	}
	if err := clientOrderOwned(o, cu); err != nil {
		return err
	}
	if req.Content == "" && req.Types == "" {
		return errs.BadRequest("请填写修改意见")
	}
	return s.DeliveryRepo.UpdateItem(ctx, cu.CompanyID, itemID, map[string]interface{}{
		"feedback_content":  req.Content,
		"feedback_types":    req.Types,
		"feedback_priority": orDefault(req.Priority, "normal"),
		"feedback_status":   1,
	})
}

// ---------------------------------------------------------------------
// 报告 H2~H4：客户侧调度费支付 / 加片费试算 / 拍摄需求修改
// ---------------------------------------------------------------------

// clientOwnedDelivery 取交付单并校验其订单归属当前客户（选片相关接口的公共前置）
func (s *Service) clientOwnedDelivery(ctx context.Context, cu *ClientUser, deliveryID int64) (*model.Delivery, *model.Order, error) {
	d, err := s.DeliveryRepo.GetByID(ctx, cu.CompanyID, deliveryID)
	if err != nil {
		return nil, nil, errs.NotFound(errs.ErrDeliveryNotFound)
	}
	o, err := s.OrderRepo.GetByID(ctx, cu.CompanyID, d.OrderID)
	if err != nil {
		return nil, nil, errs.NotFound(errs.ErrOrderNotFound)
	}
	if err := clientOrderOwned(o, cu); err != nil {
		return nil, nil, err
	}
	return d, o, nil
}

// ClientExtraQuote 加片费试算（H3：原型 C13「24/20 张 +¥240」）。
// 纯计算、不落库：客户勾选过程中实时看到超出张数与加片费，确认前金额透明。
// selectCount 传 0 时按当前已选张数试算；正式计价仍在 ClientSelectPhotos（提交选片时落库）。
func (s *Service) ClientExtraQuote(ctx context.Context, cu *ClientUser, deliveryID int64, selectCount int) (*dto.ClientExtraQuoteResp, error) {
	d, o, err := s.clientOwnedDelivery(ctx, cu, deliveryID)
	if err != nil {
		return nil, err
	}
	count := selectCount
	if count <= 0 {
		count = d.SelectedCount
	}
	included, unitPrice := 0, 0.0
	if pkg, err := s.PackageRepo.GetByID(ctx, cu.CompanyID, o.PackageID); err == nil && pkg != nil {
		included = pkg.PhotosIncluded
		unitPrice = pkg.AddonUnitPrice
	}
	extraCount, extraFee := domain.ExtraRetouchFee(count, included, unitPrice)
	return &dto.ClientExtraQuoteResp{
		IncludedCount:  included,
		SelectedCount:  count,
		ExtraCount:     extraCount,
		UnitPrice:      unitPrice,
		ExtraFee:       extraFee,
		ExtraConfirmed: d.ExtraConfirmed,
	}, nil
}

// rescheduleFeePaymentType 调度费收款单类型。
// 复用 biz_order_payment 承载调度费（无独立收费表）：核验链路与定金/尾款完全一致
// （客户上传凭证 → 工作室核验）。但调度费不属于订单套餐应收，故在 payment.go 的确认
// 核验分支中豁免「额度校验 / 已收累加」，否则订单已收会超过总额并污染应收口径。
const rescheduleFeePaymentType = "reschedule"

// listRescheduleFeePayments 取某改期单的调度费收款记录。
// OrderPayment 无 reschedule_id 列，以 remark 精确写入改期单号（RS-xxx）建立 1:1 关联。
func (s *Service) listRescheduleFeePayments(ctx context.Context, companyID int64, rs *model.OrderReschedule) ([]model.OrderPayment, error) {
	list, err := s.OrderRepo.ListPayments(ctx, companyID, rs.OrderID)
	if err != nil {
		return nil, err
	}
	out := make([]model.OrderPayment, 0, 1)
	for _, p := range list {
		if p.Type == rescheduleFeePaymentType && p.Remark == rs.Code {
			out = append(out, p)
		}
	}
	return out, nil
}

// getOwnedReschedule 取改期单并校验归属当前客户（改期单未回填客户 ID 时按订单归属兜底）
func (s *Service) getOwnedReschedule(ctx context.Context, cu *ClientUser, rescheduleID int64) (*model.OrderReschedule, error) {
	rs, err := s.RescheduleRepo.GetByID(ctx, cu.CompanyID, rescheduleID)
	if err != nil {
		return nil, errs.NotFound("改期单不存在")
	}
	if rs.CustomerID > 0 && rs.CustomerID == cu.CustomerID {
		return rs, nil
	}
	if o, err := s.OrderRepo.GetByID(ctx, cu.CompanyID, rs.OrderID); err == nil && o.CustomerID == cu.CustomerID {
		return rs, nil
	}
	return nil, errs.NotFound("改期单不存在")
}

// ClientRescheduleDetail 改期单详情 + 调度费支付状态（H2：原型 B2 改期调度费支付）
func (s *Service) ClientRescheduleDetail(ctx context.Context, cu *ClientUser, rescheduleID int64) (*dto.ClientRescheduleDetailResp, error) {
	rs, err := s.getOwnedReschedule(ctx, cu, rescheduleID)
	if err != nil {
		return nil, err
	}
	payments, err := s.listRescheduleFeePayments(ctx, cu.CompanyID, rs)
	if err != nil {
		return nil, err
	}
	status := dto.RescheduleFeeNoNeed
	if rs.FeeType == enum.RescheduleFeeCharged && rs.FeeAmount > 0 {
		status = dto.RescheduleFeeUnpaid
		for _, p := range payments {
			if p.Status == enum.PaymentStatusConfirmed {
				status = dto.RescheduleFeePaid
				break
			}
			if p.Status == enum.PaymentStatusPending {
				status = dto.RescheduleFeePending
			}
		}
	}
	return &dto.ClientRescheduleDetailResp{
		Reschedule: *rs,
		PayStatus:  status,
		Payments:   payments,
	}, nil
}

// ClientPayRescheduleFee 客户提交调度费支付凭证（H2），进入工作室核验队列。
// 前置：改期单已同意且确需收费；幂等：已有待核验/已核验记录时直接拒绝，避免重复上传。
func (s *Service) ClientPayRescheduleFee(ctx context.Context, cu *ClientUser, rescheduleID int64, req dto.ClientReschedulePayReq) (*model.OrderPayment, error) {
	rs, err := s.getOwnedReschedule(ctx, cu, rescheduleID)
	if err != nil {
		return nil, err
	}
	if rs.Status != enum.RescheduleStatusApproved {
		return nil, errs.BadRequest("改期申请尚未通过，暂无需支付调度费")
	}
	if rs.FeeType != enum.RescheduleFeeCharged || rs.FeeAmount <= 0 {
		return nil, errs.BadRequest("本次改期无需支付调度费")
	}
	existing, err := s.listRescheduleFeePayments(ctx, cu.CompanyID, rs)
	if err != nil {
		return nil, err
	}
	for _, p := range existing {
		switch p.Status {
		case enum.PaymentStatusPending:
			return nil, errs.BadRequest("调度费凭证已提交，请等待工作室核验")
		case enum.PaymentStatusConfirmed:
			return nil, errs.BadRequest("调度费已核验到账，无需重复支付")
		}
	}
	p := model.OrderPayment{
		TenantBase: model.TenantBase{
			Base:      model.Base{CreatedBy: cu.CustomerID, UpdatedBy: cu.CustomerID},
			CompanyID: cu.CompanyID,
		},
		OrderID:    rs.OrderID,
		Code:       domain.GenCode("PM"),
		CustomerID: cu.CustomerID,
		Type:       rescheduleFeePaymentType,
		Amount:     rs.FeeAmount,
		MethodID:   req.MethodID,
		Voucher:    req.Voucher,
		Remark:     rs.Code, // 与改期单 1:1 关联（表无 reschedule_id 列）
		Status:     enum.PaymentStatusPending,
	}
	if err := s.OrderRepo.CreatePayment(ctx, &p); err != nil {
		return nil, err
	}
	s.NotifyStaff(ctx, clientOperator(cu), 0, "finance", "改期调度费待核验",
		fmt.Sprintf("客户已提交改期单 %s 的调度费凭证（%.2f 元），请核验", rs.Code, rs.FeeAmount),
		"payment", p.ID)
	return &p, nil
}

// ClientUpdateOrderRequirement 客户修改拍摄需求（H4：原型 C10「修改需求」）。
// 白名单字段仅「地点 / 人数 / 风格 / 备注」：金额与套餐不在其列，拍摄日期与时段也不在——
// 日期时段变更必须走改期单（需重排档期锁），否则会出现「订单已改、档期仍锁在旧日期」。
// 仅「待定金 / 待拍摄」可改；空值表示「不修改」，避免误清空既有需求。
func (s *Service) ClientUpdateOrderRequirement(ctx context.Context, cu *ClientUser, orderID int64, req dto.ClientOrderRequirementReq) error {
	o, err := s.OrderRepo.GetByID(ctx, cu.CompanyID, orderID)
	if err != nil {
		return errs.NotFound(errs.ErrOrderNotFound)
	}
	if err := clientOrderOwned(o, cu); err != nil {
		return err
	}
	if o.Status != enum.OrderStatusPendingDeposit && o.Status != enum.OrderStatusPendingShoot {
		return errs.BadRequest("订单已进入拍摄流程，需求变更请联系工作室")
	}
	updates := map[string]interface{}{"updated_by": cu.CustomerID}
	if v := strings.TrimSpace(req.ShootAddress); v != "" {
		updates["shoot_address"] = v
	}
	if v := strings.TrimSpace(req.PeopleCount); v != "" {
		updates["people_count"] = v
	}
	if v := strings.TrimSpace(req.ShootStyle); v != "" {
		updates["shoot_style"] = v
	}
	if v := strings.TrimSpace(req.Remark); v != "" {
		updates["remark"] = v
	}
	if len(updates) == 1 {
		return errs.BadRequest("请至少填写一项需要修改的需求")
	}
	if err := s.OrderRepo.Update(ctx, cu.CompanyID, orderID, updates); err != nil {
		return err
	}
	// 需求直接影响拍前准备，必须留订单日志并提醒负责人
	op := clientOperator(cu)
	_ = s.writeOrderLog(ctx, orderID, "update_requirement", o.Status, o.Status, "客户修改拍摄需求", op)
	s.NotifyStaff(ctx, op, o.OwnerID, "order", "客户修改了拍摄需求",
		"订单 "+o.Code+" 的拍摄需求已由客户更新，请核对拍前准备", "order", o.ID)
	return nil
}
