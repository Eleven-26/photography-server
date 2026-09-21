// 交付（由 staff.go 按业务域拆分，2026-09-21 结构整改）。
package wechat

import (
	"photography-server/internal/middleware"
	"photography-server/internal/pkg/response"
	"photography-server/internal/presentation/bind"

	"github.com/gin-gonic/gin"
)

// DeliveryCreate 创建交付单
// @Summary      创建交付单
// @Description  **端差异实现**：员工端只接 body.order_id（不传 stage），与 PC 的 /delivery/create 契约不同，两个前端各自依赖，不可合并。
// @Tags         员工端·交付
// @Accept       json
// @Produce      json
// @Param        req  body  object{order_id=int}  true  "订单ID"
// @Success      200  {object}  response.Body{data=model.Delivery}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /wechat/staff/delivery/create [post]
func (h *Controller) DeliveryCreate(c *gin.Context) {
	op := middleware.GetOperator(c)
	orderID, err := bind.BodyID(c, "order_id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	d, err := h.Svc.CreateDelivery(c.Request.Context(), op, orderID)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, d)
}

// DeliverySendFinal 发送最终确认（:id 为**交付单 ID**，与 delivery/confirm 同口径）。
// PC 交付工作台没有这个动作，属移动端独有，故不进公共路由表。
// @Summary      发送最终确认
// @Description  向客户发送最终确认。⚠️ :id 为**交付单 ID**（与 /delivery/confirm 同口径）。
// @Description  PC 交付工作台无此动作，属移动端独有能力，故不进公共路由表。
// @Tags         员工端·交付
// @Accept       json
// @Produce      json
// @Param        id  path  int  true  "交付单ID"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /wechat/staff/delivery/send-final/{id} [post]
func (h *Controller) DeliverySendFinal(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.SendFinalToCustomer(c.Request.Context(), op, id); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}
