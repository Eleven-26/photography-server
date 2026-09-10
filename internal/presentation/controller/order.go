package controller

import (
	"github.com/gin-gonic/gin"

	"photography-server/internal/enum"
	"photography-server/internal/middleware"
	"photography-server/internal/pkg/errs"
	"photography-server/internal/pkg/params"
	"photography-server/internal/presentation/dto"
	"photography-server/internal/presentation/response"
)

func (h *Controller) OrderCreate(c *gin.Context) {
	op := middleware.GetOperator(c)
	var req dto.OrderCreateReq
	if err := h.bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	order, err := h.Svc.CreateOrder(c.Request.Context(), op, req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, order)
}

func (h *Controller) OrderList(c *gin.Context) {
	op := middleware.GetOperator(c)
	page, pageSize := pager(c)
	customerID := params.Int64(c, "customer_id")
	list, total, err := h.Svc.ListOrders(c.Request.Context(), op, page, pageSize,
		queryStr(c, "status"), customerID)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.PageOK(c, list, total, page, pageSize)
}

func (h *Controller) OrderDetail(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := pathID(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	detail, err := h.Svc.GetOrderDetail(c.Request.Context(), op, id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, detail)
}

func (h *Controller) OrderUpdate(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := pathID(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req dto.OrderUpdateReq
	if err := h.bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.UpdateOrder(c.Request.Context(), op, id, req); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

// OrderStatus 订单状态流转，body: {status, content}
func (h *Controller) OrderStatus(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := pathID(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req dto.OrderStatusReq
	if err := h.bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.ChangeOrderStatus(c.Request.Context(), op, id, enum.OrderStatus(req.Status), req.Content); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

func (h *Controller) OrderCancel(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := pathID(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req struct {
		Reason string `json:"reason"`
	}
	if err := h.bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.CancelOrder(c.Request.Context(), op, id, req.Reason); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

func (h *Controller) OrderLogs(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := pathID(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	detail, err := h.Svc.GetOrderDetail(c.Request.Context(), op, id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, detail.Logs)
}

// -------- 收款 --------

func (h *Controller) PaymentCreate(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := pathParam(c, "order_id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req dto.PaymentCreateReq
	if err := h.bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	p, err := h.Svc.CreatePayment(c.Request.Context(), op, id, req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, p)
}

func (h *Controller) PaymentList(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := pathParam(c, "order_id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	list, err := h.Svc.ListPayments(c.Request.Context(), op, id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, list)
}

func (h *Controller) PaymentConfirm(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := pathID(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.ConfirmPayment(c.Request.Context(), op, id); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

// PaymentDelete 删除收款记录（仅允许删除未确认的收款，已确认需走退款）
func (h *Controller) PaymentDelete(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := pathID(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.DeletePayment(c.Request.Context(), op, id); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

// -------- 退款 --------

func (h *Controller) RefundApply(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := pathParam(c, "order_id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req dto.RefundCreateReq
	if err := h.bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	r, err := h.Svc.CreateRefund(c.Request.Context(), op, id, req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, r)
}

func (h *Controller) RefundList(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := pathParam(c, "order_id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	list, err := h.Svc.ListRefunds(c.Request.Context(), op, id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, list)
}

// RefundAudit 退款审批，body: {approved, remark}
// 注意（连带修复）：审批结论必须用 *bool —— `bool + binding:"required"` 会把
// 「驳回(false)」判为未传参直接 400，导致只能通过、不能驳回；字段名同时与前端
// auditRefund(id, {approved}) 对齐（原后端读 approve，前端传 approved，恒不匹配）。
func (h *Controller) RefundAudit(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := pathID(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req struct {
		Approved *bool  `json:"approved"`
		Remark   string `json:"remark"`
	}
	if err := h.bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	if req.Approved == nil {
		response.Fail(c, errs.BadRequest("缺少审批结论 approved"))
		return
	}
	if err := h.Svc.AuditRefund(c.Request.Context(), op, id, *req.Approved, req.Remark); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

// -------- 订单加项 --------
// 加项金额计入订单总额与尾款，增删改均由 service 在同一事务内重算订单金额并写日志。

func (h *Controller) OrderAddonList(c *gin.Context) {
	op := middleware.GetOperator(c)
	orderID, err := pathParam(c, "order_id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	list, err := h.Svc.ListOrderAddons(c.Request.Context(), op, orderID)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, list)
}

func (h *Controller) OrderAddonCreate(c *gin.Context) {
	op := middleware.GetOperator(c)
	orderID, err := pathParam(c, "order_id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req dto.OrderAddonReq
	if err := h.bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	a, err := h.Svc.CreateOrderAddon(c.Request.Context(), op, orderID, req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, a)
}

func (h *Controller) OrderAddonUpdate(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := pathID(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req dto.OrderAddonReq
	if err := h.bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	a, err := h.Svc.UpdateOrderAddon(c.Request.Context(), op, id, req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, a)
}

func (h *Controller) OrderAddonDelete(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := pathID(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.DeleteOrderAddon(c.Request.Context(), op, id); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

// -------- 改期 --------
// PC 端不直接改订单拍摄日期（会漏掉档期锁重排），统一走改期单链路。

func (h *Controller) OrderRescheduleList(c *gin.Context) {
	op := middleware.GetOperator(c)
	orderID, err := pathParam(c, "order_id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	list, err := h.Svc.ListOrderReschedules(c.Request.Context(), op, orderID)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, list)
}

func (h *Controller) OrderRescheduleApply(c *gin.Context) {
	op := middleware.GetOperator(c)
	orderID, err := pathParam(c, "order_id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req dto.RescheduleApplyReq
	if err := h.bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	r, err := h.Svc.ApplyOrderReschedule(c.Request.Context(), op, orderID, req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, r)
}

// OrderRescheduleAudit 改期审批，body: {approved, remark}（同样用 *bool，避免驳回被误判未传参）
func (h *Controller) OrderRescheduleAudit(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := pathID(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req struct {
		Approved *bool  `json:"approved"`
		Remark   string `json:"remark"`
	}
	if err := h.bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	if req.Approved == nil {
		response.Fail(c, errs.BadRequest("缺少审批结论 approved"))
		return
	}
	if err := h.Svc.StaffRescheduleAudit(c.Request.Context(), op, id, *req.Approved, req.Remark); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}
