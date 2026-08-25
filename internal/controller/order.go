package controller

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"photography-server/internal/middleware"
	"photography-server/internal/response"
	"photography-server/internal/service"
)

func (h *Controller) OrderCreate(c *gin.Context) {
	op := middleware.GetOperator(c)
	var req service.OrderCreateReq
	if err := h.bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	order, err := h.Svc.CreateOrder(op, req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, order)
}

func (h *Controller) OrderList(c *gin.Context) {
	op := middleware.GetOperator(c)
	page, pageSize := pager(c)
	customerID, _ := strconv.ParseInt(c.Query("customer_id"), 10, 64)
	photographerID, _ := strconv.ParseInt(c.Query("photographer_id"), 10, 64)
	list, total, err := h.Svc.ListOrders(op, page, pageSize,
		queryStr(c, "keyword"), queryStr(c, "status"), customerID, photographerID, queryStr(c, "date"))
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
	detail, err := h.Svc.GetOrder(op, id)
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
	var req service.OrderUpdateReq
	if err := h.bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.UpdateOrder(op, id, req); err != nil {
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
	var req struct {
		Status  string `json:"status" binding:"required"`
		Content string `json:"content"`
	}
	if err := h.bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.ChangeOrderStatus(op, id, req.Status, req.Content); err != nil {
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
	if err := h.Svc.CancelOrder(op, id, req.Reason); err != nil {
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
	detail, err := h.Svc.GetOrder(op, id)
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
	var req service.PaymentCreateReq
	if err := h.bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	p, err := h.Svc.CreatePayment(op, id, req)
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
	list, err := h.Svc.ListPayments(op, id)
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
	if err := h.Svc.ConfirmPayment(op, id); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

func (h *Controller) PaymentDelete(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := pathID(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.DeletePayment(op, id); err != nil {
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
	var req service.RefundCreateReq
	if err := h.bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	r, err := h.Svc.ApplyRefund(op, id, req)
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
	list, err := h.Svc.ListRefunds(op, id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, list)
}

func (h *Controller) RefundAudit(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := pathID(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req struct {
		Approve bool   `json:"approve" binding:"required"`
		Remark  string `json:"remark"`
	}
	if err := h.bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.AuditRefund(op, id, req.Approve, req.Remark); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}
