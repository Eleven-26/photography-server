package controller

import (
	"github.com/gin-gonic/gin"

	"photography-server/internal/middleware"
	"photography-server/internal/pkg/response"
	"photography-server/internal/presentation/bind"
)

// @Summary      财务汇总
// @Description  按月汇总收款 / 退款 / 待确认金额等指标，缺省当月。
// @Tags         财务
// @Accept       json
// @Produce      json
// @Param        req  body  object{month=string}  true  "查询条件（month 格式 2006-01，缺省当月）"
// @Success      200  {object}  response.Body{data=contract.FinanceSummaryResp}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /finance/summary [post]
func (h *Controller) FinanceSummary(c *gin.Context) {
	op := middleware.GetOperator(c)
	sum, err := h.Svc.FinanceSummary(c.Request.Context(), op, bind.ParamStr(c, "month"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, sum)
}

// @Summary      财务收款流水
// @Description  分页查询收款流水，可按确认状态过滤。
// @Tags         财务
// @Accept       json
// @Produce      json
// @Param        req  body  object{page=int,page_size=int,keyword=string,status=string}  true  "查询条件（status 收款确认状态，空=全部）"
// @Success      200  {object}  response.Body{data=response.Page{list=[]model.OrderPayment,total=int,page=int,page_size=int}}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /finance/payments [post]
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

// @Summary      财务退款流水
// @Description  分页查询退款流水，可按审批状态过滤。
// @Tags         财务
// @Accept       json
// @Produce      json
// @Param        req  body  object{page=int,page_size=int,keyword=string,status=string}  true  "查询条件（status 退款状态，空=全部）"
// @Success      200  {object}  response.Body{data=response.Page{list=[]model.OrderRefund,total=int,page=int,page_size=int}}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /finance/refunds [post]
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
// @Summary      导出对账 CSV
// @Description  按月份导出对账明细，缺省当月。**响应为 CSV 文件流，浏览器直接下载，不走统一 JSON 包装**。
// @Description  权限点独立于 finance:view —— 可见不等于可带走（finance:export）。
// @Tags         财务
// @Accept       json
// @Produce      text/csv
// @Param        req  body  object{month=string}  true  "查询条件（month 格式 2006-01，缺省当月）"
// @Success      200  {string}  string  "CSV 文件内容（Content-Disposition 附带文件名）"
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /finance/export [post]
func (h *Controller) FinanceExport(c *gin.Context) {
	op := middleware.GetOperator(c)
	filename, content, err := h.Svc.FinanceExport(c.Request.Context(), op, bind.ParamStr(c, "month"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.File(c, filename, "text/csv; charset=utf-8", content)
}
