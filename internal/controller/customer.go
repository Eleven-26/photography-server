package controller

import (
	"github.com/gin-gonic/gin"

	"photography-server/internal/middleware"
	"photography-server/internal/response"
	"photography-server/internal/service"
)

func (h *Controller) CustomerList(c *gin.Context) {
	op := middleware.GetOperator(c)
	page, pageSize := pager(c)
	list, total, err := h.Svc.ListCustomers(op, page, pageSize,
		queryStr(c, "keyword"), queryStr(c, "status"), queryStr(c, "level"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.PageOK(c, list, total, page, pageSize)
}

func (h *Controller) CustomerDetail(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := pathID(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	detail, err := h.Svc.GetCustomer(op, id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, detail)
}

func (h *Controller) CustomerCreate(c *gin.Context) {
	op := middleware.GetOperator(c)
	var req service.CustomerCreateReq
	if err := h.bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	customer, err := h.Svc.CreateCustomer(op, req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, customer)
}

func (h *Controller) CustomerUpdate(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := pathID(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req service.CustomerUpdateReq
	if err := h.bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.UpdateCustomer(op, id, req); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

func (h *Controller) CustomerDelete(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := pathID(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.DeleteCustomer(op, id); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

func (h *Controller) CustomerStats(c *gin.Context) {
	op := middleware.GetOperator(c)
	st, err := h.Svc.CustomerStats(op)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, st)
}

// CustomerOrders 客户名下订单
func (h *Controller) CustomerOrders(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := pathID(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	page, pageSize := pager(c)
	list, total, err := h.Svc.ListOrders(op, page, pageSize, "", "", id, 0, "")
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.PageOK(c, list, total, page, pageSize)
}
