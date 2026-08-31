package controller

import (
	"github.com/gin-gonic/gin"

	"photography-server/internal/middleware"
	"photography-server/internal/response"
)

func (h *Controller) FinanceSummary(c *gin.Context) {
	op := middleware.GetOperator(c)
	sum, err := h.Svc.FinanceSummary(op, queryStr(c, "month"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, sum)
}

func (h *Controller) FinancePayments(c *gin.Context) {
	op := middleware.GetOperator(c)
	page, pageSize := pager(c)
	list, total, err := h.Svc.ListFinancePayments(op, page, pageSize,
		queryStr(c, "start_date"), queryStr(c, "end_date"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.PageOK(c, list, total, page, pageSize)
}

func (h *Controller) FinanceRefunds(c *gin.Context) {
	op := middleware.GetOperator(c)
	page, pageSize := pager(c)
	list, total, err := h.Svc.ListFinanceRefunds(op, page, pageSize,
		queryStr(c, "start_date"), queryStr(c, "end_date"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.PageOK(c, list, total, page, pageSize)
}
