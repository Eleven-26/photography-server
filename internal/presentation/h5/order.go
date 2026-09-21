// 预约下单与订单（由 h5.go 按业务域拆分，2026-09-21 结构整改）。
package h5

import (
	"photography-server/internal/contract"
	"photography-server/internal/middleware"
	"photography-server/internal/pkg/params"
	"photography-server/internal/pkg/response"
	"photography-server/internal/presentation/bind"

	"github.com/gin-gonic/gin"
)

// BookingSubmit 提交预约单。
// 分享人归属：链接参数 staff_id（头/body/query 三选一，见 staffFrom）随预约一并落到订单，
// 客户从谁的预约主页进来下单，订单就算谁的。
// @Summary      提交预约单
// @Description  客户提交预约单。分享人归属：链接参数 staff_id 随预约落到订单（biz_order.photographer_id）。
// @Tags         客户区·订单
// @Accept       json
// @Produce      json
// @Param        req  body  contract.ClientBookingReq  true  "预约信息"
// @Success      200  {object}  response.Body{data=model.Order}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /h5/order/submit [post]
func (h *Controller) BookingSubmit(c *gin.Context) {
	cu := middleware.GetClientUser(c)
	var req contract.ClientBookingReq
	if err := bind.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	o, err := h.Svc.ClientSubmitBooking(c.Request.Context(), cu, req, staffFrom(c))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, o)
}

// BookingConfirm 确认预约单
// @Summary      确认预约单
// @Tags         客户区·订单
// @Accept       json
// @Produce      json
// @Param        id  path  int  true  "订单ID"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /h5/order/confirm/{id} [post]
func (h *Controller) BookingConfirm(c *gin.Context) {
	cu := middleware.GetClientUser(c)
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.ClientConfirmBooking(c.Request.Context(), cu, id); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

// BookingCancel 取消预约单
// @Summary      取消预约单
// @Description  取消本人的预约单并记录原因；body 可为空（reason 可选）。
// @Tags         客户区·订单
// @Accept       json
// @Produce      json
// @Param        id   path  int                   true  "订单ID"
// @Param        req  body  object{reason=string} false  "取消原因（可省略）"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /h5/order/cancel/{id} [post]
func (h *Controller) BookingCancel(c *gin.Context) {
	cu := middleware.GetClientUser(c)
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req struct {
		Reason string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&req)
	if err := h.Svc.ClientCancelBooking(c.Request.Context(), cu, id, req.Reason); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

// OrderList 我的订单
// @Summary      我的订单
// @Description  分页查询当前客户名下订单，可按状态过滤（归属由令牌内 customer_id 锁定）。
// @Tags         客户区·订单
// @Accept       json
// @Produce      json
// @Param        req  body  object{page=int,page_size=int,status=string}  true  "查询条件"
// @Success      200  {object}  response.Body{data=response.Page{list=[]model.Order,total=int,page=int,page_size=int}}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /h5/order/list [post]
func (h *Controller) OrderList(c *gin.Context) {
	cu := middleware.GetClientUser(c)
	page, pageSize := bind.Pager(c)
	list, total, err := h.Svc.ClientOrders(c.Request.Context(), cu, page, pageSize, params.Str(c, "status"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.PageOK(c, list, total, page, pageSize)
}

// OrderDetail 订单详情
// @Summary      订单详情
// @Tags         客户区·订单
// @Accept       json
// @Produce      json
// @Param        id  path  int  true  "订单ID"
// @Success      200  {object}  response.Body{data=contract.ClientOrderDetail}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /h5/order/detail/{id} [post]
func (h *Controller) OrderDetail(c *gin.Context) {
	cu := middleware.GetClientUser(c)
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	detail, err := h.Svc.ClientOrderDetail(c.Request.Context(), cu, id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, detail)
}

// OrderPrepRead 客户确认已读「拍前准备清单」（写 biz_order.prep_read_at，幂等）
// @Summary      拍前准备已读
// @Description  客户确认已读「拍前准备清单」，写 biz_order.prep_read_at；**幂等**。
// @Tags         客户区·订单
// @Accept       json
// @Produce      json
// @Param        id  path  int  true  "订单ID"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /h5/order/prep/read/{id} [post]
func (h *Controller) OrderPrepRead(c *gin.Context) {
	cu := middleware.GetClientUser(c)
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.ClientReadOrderPrep(c.Request.Context(), cu, id); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

// OrderRequirementUpdate 客户修改拍摄需求（仅待定金/待拍摄，白名单字段）
// @Summary      修改拍摄需求
// @Description  客户修改拍摄需求，**仅待定金 / 待拍摄状态可改，且只接受白名单字段**。
// @Tags         客户区·订单
// @Accept       json
// @Produce      json
// @Param        id   path  int                                  true  "订单ID"
// @Param        req  body  contract.ClientOrderRequirementReq    true  "需求字段（白名单）"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /h5/order/requirement/update/{id} [post]
func (h *Controller) OrderRequirementUpdate(c *gin.Context) {
	cu := middleware.GetClientUser(c)
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req contract.ClientOrderRequirementReq
	if err := bind.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.ClientUpdateOrderRequirement(c.Request.Context(), cu, id, req); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}
