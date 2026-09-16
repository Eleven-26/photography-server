package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"photography-server/internal/contract"
	"photography-server/internal/enum"
	"photography-server/internal/model"
	"photography-server/internal/pkg/errs"
	"photography-server/internal/pkg/logger"
)

// orderRemarkMaxRunes 订单备注列（varchar(500)）的安全上限，留出余量按 rune 截断。
const orderRemarkMaxRunes = 480

// ConvertCustomRequestToOrder 把定制需求转为订单（管理端手动操作）。
//
// 定制需求是「非套餐」诉求，转单必须先选定套餐，因此请求体直接复用 OrderCreateReq
// （package_id 必填），建单全流程复用 CreateOrder —— 金额拆分、客户快照、日历占位、
// 客户消费累计都只有一份实现，避免转单路径与常规下单产生口径漂移。
//
// 客户解析优先级：请求显式 customer_id > 需求已关联的 customer_id > 按需求手机号查档
// （查不到则建档，见 FindOrCreateCustomerByMobile）。
//
// 建单成功后把需求标记为「已响应」并写入转单说明，避免同一需求被重复转单。
func (s *Service) ConvertCustomRequestToOrder(ctx context.Context, op Operator, id int64, req contract.OrderCreateReq) (*model.Order, error) {
	if req.PackageID <= 0 {
		return nil, errs.BadRequest(errs.ErrCustomRequestPackageRequired)
	}
	cr, err := s.CustomRequestRepo.GetByID(ctx, op.CompanyID, id)
	if err != nil {
		return nil, errs.NotFound(errs.ErrCustomRequestNotFound)
	}
	if cr.Status == enum.CustomRequestClosed {
		return nil, errs.BadRequest(errs.ErrCustomRequestClosed)
	}

	if req.CustomerID == 0 {
		req.CustomerID = cr.CustomerID
	}
	if req.CustomerID == 0 {
		c, err := s.FindOrCreateCustomerByMobile(ctx, op.CompanyID, cr.Name, cr.Mobile, "定制需求")
		if err != nil {
			return nil, err
		}
		if c != nil {
			req.CustomerID = c.ID
			// 需求补挂客户，后续订单/交付/通知可按客户主体聚合
			if err := s.CustomRequestRepo.Update(ctx, op.CompanyID, id, map[string]interface{}{
				"customer_id": c.ID,
				"updated_by":  op.UserID,
			}); err != nil {
				logger.Warnf("ConvertCustomRequestToOrder: 需求补挂客户失败 id=%d customerID=%d err=%v", id, c.ID, err)
			}
		}
	}
	if strings.TrimSpace(req.Remark) == "" {
		req.Remark = customRequestRemark(cr)
	}

	o, err := s.CreateOrder(ctx, op, req)
	if err != nil {
		return nil, err
	}

	updates := map[string]interface{}{
		"status":      enum.CustomRequestResponded,
		"response":    "已转为订单 " + o.Code,
		"response_by": op.UserID,
		"response_at": time.Now().Format("2006-01-02 15:04:05"),
		"updated_by":  op.UserID,
	}
	// 认领落店：与 StaffCustomRequestRespond 同口径（公共池需求被处理后归属操作人门店）
	if cr.StoreID == 0 && op.StoreID != 0 {
		updates["store_id"] = op.StoreID
	}
	if err := s.CustomRequestRepo.Update(ctx, op.CompanyID, id, updates); err != nil {
		// 订单已落地，此处失败不回滚：否则用户看到"转单失败"却已产生订单，重试即重复建单。
		logger.Warnf("ConvertCustomRequestToOrder: 回写定制需求失败 id=%d orderID=%d err=%v", id, o.ID, err)
	}
	return o, nil
}

// customRequestRemark 把定制需求诉求压成订单备注，保留来源链路，避免转单后需求信息丢失。
func customRequestRemark(cr *model.CustomRequest) string {
	parts := []string{"来源：定制需求"}
	if cr.ProjectType != "" {
		parts = append(parts, "类型："+cr.ProjectType)
	}
	if cr.ExpectedDate != "" {
		parts = append(parts, "期望日期："+cr.ExpectedDate)
	}
	if cr.Location != "" {
		parts = append(parts, "期望地点："+cr.Location)
	}
	if cr.BudgetMin > 0 || cr.BudgetMax > 0 {
		parts = append(parts, fmt.Sprintf("预算：%.0f-%.0f", cr.BudgetMin, cr.BudgetMax))
	}
	if cr.Detail != "" {
		parts = append(parts, "需求："+cr.Detail)
	}
	out := strings.Join(parts, "；")
	if r := []rune(out); len(r) > orderRemarkMaxRunes {
		out = string(r[:orderRemarkMaxRunes])
	}
	return out
}
