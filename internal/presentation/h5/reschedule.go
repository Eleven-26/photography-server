// 改期（由 h5.go 按业务域拆分，2026-09-21 结构整改）。
package h5

import (
	"photography-server/internal/contract"
	"photography-server/internal/middleware"
	"photography-server/internal/pkg/response"
	"photography-server/internal/presentation/bind"

	"github.com/gin-gonic/gin"
)

// RescheduleApply 申请改期
// @Summary      申请改期
// @Description  客户发起改期申请（apply_source=2）。
// @Tags         客户区·改期
// @Accept       json
// @Produce      json
// @Param        order_id  path  int                         true  "订单ID"
// @Param        req       body  contract.ClientRescheduleReq true  "改期信息"
// @Success      200  {object}  response.Body{data=model.OrderReschedule}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /h5/reschedule/apply/{order_id} [post]
func (h *Controller) RescheduleApply(c *gin.Context) {
	cu := middleware.GetClientUser(c)
	orderID, err := bind.PathID(c, "order_id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req contract.ClientRescheduleReq
	if err := bind.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	rs, err := h.Svc.ClientRescheduleApply(c.Request.Context(), cu, orderID, req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, rs)
}

// RescheduleCancel 撤回改期申请
// @Summary      撤回改期申请
// @Tags         客户区·改期
// @Accept       json
// @Produce      json
// @Param        id  path  int  true  "改期单ID"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /h5/reschedule/cancel/{id} [post]
func (h *Controller) RescheduleCancel(c *gin.Context) {
	cu := middleware.GetClientUser(c)
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.ClientRescheduleCancel(c.Request.Context(), cu, id); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

// RescheduleList 我的订单改期单列表（改期进度页；不分页，逐单明细集合）
// @Summary      我的改期单列表
// @Description  返回某订单下的改期单（改期进度页；**不分页**，逐单明细集合）。
// @Tags         客户区·改期
// @Accept       json
// @Produce      json
// @Param        order_id  path  int  true  "订单ID"
// @Success      200  {object}  response.Body{data=[]model.OrderReschedule}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /h5/reschedule/list/{order_id} [post]
func (h *Controller) RescheduleList(c *gin.Context) {
	cu := middleware.GetClientUser(c)
	orderID, err := bind.PathID(c, "order_id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	list, err := h.Svc.ClientReschedules(c.Request.Context(), cu, orderID)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, list)
}

// RescheduleDetail 改期单详情 + 调度费支付状态
// @Summary      改期单详情
// @Description  改期单详情，含调度费支付状态。
// @Tags         客户区·改期
// @Accept       json
// @Produce      json
// @Param        id  path  int  true  "改期单ID"
// @Success      200  {object}  response.Body{data=contract.ClientRescheduleDetailResp}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /h5/reschedule/detail/{id} [post]
func (h *Controller) RescheduleDetail(c *gin.Context) {
	cu := middleware.GetClientUser(c)
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	d, err := h.Svc.ClientRescheduleDetail(c.Request.Context(), cu, id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, d)
}

// ReschedulePay 提交改期调度费支付凭证
// @Summary      提交调度费凭证
// @Description  提交改期调度费的支付凭证（资金不经平台，走「上传凭证 → 工作室核验」）。
// @Tags         客户区·改期
// @Accept       json
// @Produce      json
// @Param        id   path  int                               true  "改期单ID"
// @Param        req  body  contract.ClientReschedulePayReq    true  "支付凭证"
// @Success      200  {object}  response.Body{data=model.OrderPayment}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /h5/reschedule/pay/{id} [post]
func (h *Controller) ReschedulePay(c *gin.Context) {
	cu := middleware.GetClientUser(c)
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req contract.ClientReschedulePayReq
	if err := bind.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	p, err := h.Svc.ClientPayRescheduleFee(c.Request.Context(), cu, id, req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, p)
}
