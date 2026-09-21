// 收款登记与收款方式（由 h5.go 按业务域拆分，2026-09-21 结构整改）。
package h5

import (
	"photography-server/internal/contract"
	"photography-server/internal/middleware"
	"photography-server/internal/pkg/response"
	"photography-server/internal/presentation/bind"

	"github.com/gin-gonic/gin"
)

// PaymentList 我的订单收款记录（支付页展示登记状态）
// @Summary      我的收款记录
// @Description  返回某订单的收款记录（支付页展示登记状态；**不分页**）。
// @Tags         客户区·收款
// @Accept       json
// @Produce      json
// @Param        order_id  path  int  true  "订单ID"
// @Success      200  {object}  response.Body{data=[]model.OrderPayment}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /h5/payment/list/{order_id} [post]
func (h *Controller) PaymentList(c *gin.Context) {
	cu := middleware.GetClientUser(c)
	orderID, err := bind.PathID(c, "order_id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	list, err := h.Svc.ClientPayments(c.Request.Context(), cu, orderID)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, list)
}

// PaymentMark 客户登记转账（「我已完成转账，通知摄影师」）。
// 资金不经平台：仅落 status=1 待核验记录，到账确认仍在员工端 /payment/confirm/:id。
// @Summary      登记转账
// @Description  客户登记「我已完成转账，通知摄影师」。资金不经平台：仅落一条待核验记录，到账确认仍在员工端 /payment/confirm/{id}。
// @Tags         客户区·收款
// @Accept       json
// @Produce      json
// @Param        req  body  contract.ClientPaymentMarkReq  true  "转账登记（含 order_id）"
// @Success      200  {object}  response.Body{data=model.OrderPayment}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /h5/pay/mark [post]
func (h *Controller) PaymentMark(c *gin.Context) {
	cu := middleware.GetClientUser(c)
	var req contract.ClientPaymentMarkReq
	if err := bind.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	p, err := h.Svc.ClientRegisterPayment(c.Request.Context(), cu, req.OrderID, req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, p)
}

// PaymentMethods 客户可见的收款方式（只出启用项，供支付页展示收款码/账号）
// @Summary      收款方式列表
// @Description  客户可见的收款方式，**只返回启用项**，供支付页展示收款码/账号。
// @Tags         客户区·收款
// @Accept       json
// @Produce      json
// @Success      200  {object}  response.Body{data=[]contract.ClientPaymentMethodResp}
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /h5/payment-method/list [post]
func (h *Controller) PaymentMethods(c *gin.Context) {
	cu := middleware.GetClientUser(c)
	list, err := h.Svc.ClientPaymentMethods(c.Request.Context(), cu)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, list)
}
