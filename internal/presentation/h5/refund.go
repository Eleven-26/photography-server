// 退款（由 h5.go 按业务域拆分，2026-09-21 结构整改）。
package h5

import (
	"photography-server/internal/contract"
	"photography-server/internal/middleware"
	"photography-server/internal/pkg/response"
	"photography-server/internal/presentation/bind"

	"github.com/gin-gonic/gin"
)

// RefundApply 申请退款
// @Summary      申请退款
// @Description  客户对订单发起退款申请；金额为空时按退款规则自动计算。
// @Tags         客户区·退款
// @Accept       json
// @Produce      json
// @Param        order_id  path  int                     true  "订单ID"
// @Param        req       body  contract.ClientRefundReq true  "退款请求"
// @Success      200  {object}  response.Body{data=model.OrderRefund}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /h5/refund/apply/{order_id} [post]
func (h *Controller) RefundApply(c *gin.Context) {
	cu := middleware.GetClientUser(c)
	orderID, err := bind.PathID(c, "order_id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req contract.ClientRefundReq
	if err := bind.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	rf, err := h.Svc.ClientRefundApply(c.Request.Context(), cu, orderID, req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, rf)
}

// RefundList 我的订单退款记录（退款进度页 C21；不分页，逐单明细集合）
// @Summary      我的退款记录
// @Description  返回某订单下的退款记录（退款进度页；**不分页**）。
// @Tags         客户区·退款
// @Accept       json
// @Produce      json
// @Param        order_id  path  int  true  "订单ID"
// @Success      200  {object}  response.Body{data=[]model.OrderRefund}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /h5/refund/list/{order_id} [post]
func (h *Controller) RefundList(c *gin.Context) {
	cu := middleware.GetClientUser(c)
	orderID, err := bind.PathID(c, "order_id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	list, err := h.Svc.ClientRefunds(c.Request.Context(), cu, orderID)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, list)
}

// RefundConfirm 客户确认收到退款（写 customer_confirm_at，与员工端审批闭环；幂等）
// @Summary      确认收到退款
// @Description  客户确认收到退款，写 customer_confirm_at，与员工端审批闭环；**幂等**。
// @Tags         客户区·退款
// @Accept       json
// @Produce      json
// @Param        id  path  int  true  "退款单ID"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /h5/refund/confirm/{id} [post]
func (h *Controller) RefundConfirm(c *gin.Context) {
	cu := middleware.GetClientUser(c)
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.ClientConfirmRefundReceived(c.Request.Context(), cu, id); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}
