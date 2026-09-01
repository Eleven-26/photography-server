package controller

import (
	"github.com/gin-gonic/gin"

	"photography-server/internal/middleware"
	"photography-server/internal/presentation/dto"
	"photography-server/internal/response"
)

func (h *Controller) Workspace(c *gin.Context) {
	op := middleware.GetOperator(c)
	w, err := h.Svc.Workspace(op)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, w)
}

func (h *Controller) CompanyUpdate(c *gin.Context) {
	op := middleware.GetOperator(c)
	var req dto.CompanyUpdateReq
	if err := h.bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.UpdateCompany(op, req); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

func (h *Controller) PaymentMethodList(c *gin.Context) {
	op := middleware.GetOperator(c)
	list, err := h.Svc.ListPaymentMethods(op)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, list)
}

func (h *Controller) PaymentMethodCreate(c *gin.Context) {
	op := middleware.GetOperator(c)
	var req dto.PaymentMethodReq
	if err := h.bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.CreatePaymentMethod(op, req); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

func (h *Controller) PaymentMethodUpdate(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := pathID(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req dto.PaymentMethodReq
	if err := h.bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.UpdatePaymentMethod(op, id, req); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

func (h *Controller) PaymentMethodDelete(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := pathID(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.DeletePaymentMethod(op, id); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

func (h *Controller) OperationLogList(c *gin.Context) {
	op := middleware.GetOperator(c)
	page, pageSize := pager(c)
	list, total, err := h.Svc.ListOperationLogs(op, page, pageSize)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.PageOK(c, list, total, page, pageSize)
}
