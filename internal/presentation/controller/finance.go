package controller

import (
	"github.com/gin-gonic/gin"

	"photography-server/internal/middleware"
	"photography-server/internal/presentation/bind"
	"photography-server/internal/presentation/response"
)

func (h *Controller) FinanceSummary(c *gin.Context) {
	op := middleware.GetOperator(c)
	sum, err := h.Svc.FinanceSummary(c.Request.Context(), op, bind.ParamStr(c, "month"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, sum)
}

func (h *Controller) FinancePayments(c *gin.Context) {
	op := middleware.GetOperator(c)
	page, pageSize := bind.Pager(c)
	list, total, err := h.Svc.ListFinancePayments(c.Request.Context(), op, page, pageSize, bind.ParamStr(c, "status"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.PageOK(c, list, total, page, pageSize)
}

func (h *Controller) FinanceRefunds(c *gin.Context) {
	op := middleware.GetOperator(c)
	page, pageSize := bind.Pager(c)
	list, total, err := h.Svc.ListFinanceRefunds(c.Request.Context(), op, page, pageSize, bind.ParamStr(c, "status"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.PageOK(c, list, total, page, pageSize)
}

// FinanceExport 导出对账 CSV（按月份，缺省当月）。浏览器直接下载，不走统一 JSON 包装。
func (h *Controller) FinanceExport(c *gin.Context) {
	op := middleware.GetOperator(c)
	filename, content, err := h.Svc.FinanceExport(c.Request.Context(), op, bind.ParamStr(c, "month"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.File(c, filename, "text/csv; charset=utf-8", content)
}
