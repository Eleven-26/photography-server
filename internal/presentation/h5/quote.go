// 报价（由 h5.go 按业务域拆分，2026-09-21 结构整改）。
package h5

import (
	"photography-server/internal/contract"
	"photography-server/internal/middleware"
	"photography-server/internal/pkg/response"
	"photography-server/internal/presentation/bind"

	"github.com/gin-gonic/gin"
)

// QuoteList 我的报价单列表（含明细字段，前端按 id 取单条即可）
// @Summary      我的报价单列表
// @Description  返回本人的报价单（含明细字段，前端按 id 取单条即可）。
// @Tags         客户区·报价
// @Accept       json
// @Produce      json
// @Success      200  {object}  response.Body{data=[]model.Quote}
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /h5/quote/list [post]
func (h *Controller) QuoteList(c *gin.Context) {
	cu := middleware.GetClientUser(c)
	list, err := h.Svc.ClientQuotes(c.Request.Context(), cu)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, list)
}

// QuoteDetail 单张报价详情（报价详情页，按 id 直取；归属校验含线索兜底）
// @Summary      报价详情
// @Description  单张报价详情，按 id 直取；归属校验含线索兜底。
// @Tags         客户区·报价
// @Accept       json
// @Produce      json
// @Param        id  path  int  true  "报价单ID"
// @Success      200  {object}  response.Body{data=model.Quote}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /h5/quote/detail/{id} [post]
func (h *Controller) QuoteDetail(c *gin.Context) {
	cu := middleware.GetClientUser(c)
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	q, err := h.Svc.ClientQuoteDetail(c.Request.Context(), cu, id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, q)
}

// QuoteAccept 接受报价
// @Summary      接受报价
// @Tags         客户区·报价
// @Accept       json
// @Produce      json
// @Param        id  path  int  true  "报价单ID"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /h5/quote/accept/{id} [post]
func (h *Controller) QuoteAccept(c *gin.Context) {
	cu := middleware.GetClientUser(c)
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.ClientQuoteAccept(c.Request.Context(), cu, id); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

// QuoteModify 对报价提出修改意见
// @Summary      对报价提修改意见
// @Tags         客户区·报价
// @Accept       json
// @Produce      json
// @Param        id   path  int                          true  "报价单ID"
// @Param        req  body  contract.ClientQuoteModifyReq true  "修改意见"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /h5/quote/modify/{id} [post]
func (h *Controller) QuoteModify(c *gin.Context) {
	cu := middleware.GetClientUser(c)
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req contract.ClientQuoteModifyReq
	if err := bind.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.ClientQuoteModify(c.Request.Context(), cu, id, req.Content); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}
