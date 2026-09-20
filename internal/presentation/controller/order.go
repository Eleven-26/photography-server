package controller

import (
	"github.com/gin-gonic/gin"

	"photography-server/internal/contract"
	"photography-server/internal/enum"
	"photography-server/internal/middleware"
	"photography-server/internal/pkg/errs"
	"photography-server/internal/pkg/params"
	"photography-server/internal/pkg/response"
	"photography-server/internal/presentation/bind"
)

// @Summary      创建订单
// @Description  创建订单；客户与线索二选一（customer_id / lead_id），套餐必填。
// @Tags         订单
// @Accept       json
// @Produce      json
// @Param        req  body  contract.OrderCreateReq  true  "订单信息"
// @Success      200  {object}  response.Body{data=model.Order}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /order/create [post]
func (h *Controller) OrderCreate(c *gin.Context) {
	op := middleware.GetOperator(c)
	var req contract.OrderCreateReq
	if err := bind.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	order, err := h.Svc.CreateOrder(c.Request.Context(), op, req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, order)
}

// @Summary      订单列表
// @Description  分页查询订单，可按状态、客户过滤。
// @Tags         订单
// @Accept       json
// @Produce      json
// @Param        req  body  contract.OrderListReq  true  "查询条件"
// @Success      200  {object}  response.Body{data=response.Page{list=[]model.Order,total=int,page=int,page_size=int}}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /order/list [post]
func (h *Controller) OrderList(c *gin.Context) {
	op := middleware.GetOperator(c)
	page, pageSize := bind.Pager(c)
	customerID := params.Int64(c, "customer_id")
	list, total, err := h.Svc.ListOrders(c.Request.Context(), op, page, pageSize,
		bind.ParamStr(c, "status"), customerID)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.PageOK(c, list, total, page, pageSize)
}

// @Summary      订单详情
// @Description  返回订单及其收款、退款、日志、交付与可流转状态。
// @Tags         订单
// @Accept       json
// @Produce      json
// @Param        id  path  int  true  "订单ID"
// @Success      200  {object}  response.Body{data=contract.OrderDetail}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /order/detail/{id} [post]
func (h *Controller) OrderDetail(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	detail, err := h.Svc.GetOrderDetail(c.Request.Context(), op, id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, detail)
}

// @Summary      更新订单
// @Description  更新订单拍摄信息；body.id 为主键。
// @Tags         订单
// @Accept       json
// @Produce      json
// @Param        req  body  contract.OrderUpdateReq  true  "订单信息（含 id）"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /order/update [post]
func (h *Controller) OrderUpdate(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := bind.BodyID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req contract.OrderUpdateReq
	if err := bind.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.UpdateOrder(c.Request.Context(), op, id, req); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

// OrderStatus 订单状态流转，body: {status, content}
// @Summary      订单状态流转
// @Description  按状态机流转订单状态，body: {status, content}。
// @Tags         订单
// @Accept       json
// @Produce      json
// @Param        id   path  int                      true  "订单ID"
// @Param        req  body  contract.OrderStatusReq  true  "状态流转请求"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /order/status/{id} [post]
func (h *Controller) OrderStatus(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req contract.OrderStatusReq
	if err := bind.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.ChangeOrderStatus(c.Request.Context(), op, id, enum.OrderStatus(req.Status), req.Content); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

// @Summary      取消订单
// @Description  取消订单并记录原因；当前状态不允许取消时返回业务错误。
// @Tags         订单
// @Accept       json
// @Produce      json
// @Param        id   path  int                     true  "订单ID"
// @Param        req  body  contract.OrderCancelReq true  "取消原因"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /order/cancel/{id} [post]
func (h *Controller) OrderCancel(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req contract.OrderCancelReq
	if err := bind.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.CancelOrder(c.Request.Context(), op, id, req.Reason); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

// @Summary      订单操作日志
// @Tags         订单
// @Accept       json
// @Produce      json
// @Param        id  path  int  true  "订单ID"
// @Success      200  {object}  response.Body{data=[]model.OrderLog}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /order/logs/{id} [post]
func (h *Controller) OrderLogs(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	detail, err := h.Svc.GetOrderDetail(c.Request.Context(), op, id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, detail.Logs)
}

// -------- 收款 --------

// @Summary      登记收款
// @Description  登记订单收款（资金不经平台，仅登记）；body.order_id 为订单主键。
// @Tags         收款
// @Accept       json
// @Produce      json
// @Param        req  body  contract.PaymentCreateReq  true  "收款信息（含 order_id）"
// @Success      200  {object}  response.Body{data=model.OrderPayment}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /payment/create [post]
func (h *Controller) PaymentCreate(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := bind.BodyID(c, "order_id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req contract.PaymentCreateReq
	if err := bind.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	p, err := h.Svc.CreatePayment(c.Request.Context(), op, id, req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, p)
}

// @Summary      订单收款列表
// @Tags         收款
// @Accept       json
// @Produce      json
// @Param        order_id  path  int  true  "订单ID"
// @Success      200  {object}  response.Body{data=[]model.OrderPayment}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /payment/list/{order_id} [post]
func (h *Controller) PaymentList(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := bind.PathID(c, "order_id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	list, err := h.Svc.ListPayments(c.Request.Context(), op, id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, list)
}

// @Summary      核验收款
// @Description  核验到账，将收款置为已确认。
// @Tags         收款
// @Accept       json
// @Produce      json
// @Param        id  path  int  true  "收款记录ID"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /payment/confirm/{id} [post]
func (h *Controller) PaymentConfirm(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.ConfirmPayment(c.Request.Context(), op, id); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

// PaymentDelete 删除收款记录（仅允许删除未确认的收款，已确认需走退款）
// @Summary      删除收款
// @Description  删除未确认的收款记录；已确认的收款需走退款流程。
// @Tags         收款
// @Accept       json
// @Produce      json
// @Param        id  path  int  true  "收款记录ID"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /payment/delete/{id} [post]
func (h *Controller) PaymentDelete(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.DeletePayment(c.Request.Context(), op, id); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

// -------- 退款 --------

// @Summary      申请退款
// @Description  对指定订单发起退款；金额为空时按退款规则自动计算。
// @Tags         退款
// @Accept       json
// @Produce      json
// @Param        order_id  path  int                      true  "订单ID"
// @Param        req       body  contract.RefundCreateReq  true  "退款请求"
// @Success      200  {object}  response.Body{data=model.OrderRefund}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /refund/apply/{order_id} [post]
func (h *Controller) RefundApply(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := bind.PathID(c, "order_id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req contract.RefundCreateReq
	if err := bind.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	r, err := h.Svc.CreateRefund(c.Request.Context(), op, id, req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, r)
}

// @Summary      订单退款列表
// @Tags         退款
// @Accept       json
// @Produce      json
// @Param        order_id  path  int  true  "订单ID"
// @Success      200  {object}  response.Body{data=[]model.OrderRefund}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /refund/list/{order_id} [post]
func (h *Controller) RefundList(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := bind.PathID(c, "order_id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	list, err := h.Svc.ListRefunds(c.Request.Context(), op, id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, list)
}

// RefundAudit 退款审批，body: {approved, remark}
// 注意（连带修复）：审批结论必须用 *bool —— `bool + binding:"required"` 会把
// 「驳回(false)」判为未传参直接 400，导致只能通过、不能驳回；字段名同时与前端
// auditRefund(id, {approved}) 对齐（原后端读 approve，前端传 approved，恒不匹配）。
// @Summary      退款审批
// @Description  审批退款；approved 为 true 通过、false 驳回，缺省返回参数错误。
// @Tags         退款
// @Accept       json
// @Produce      json
// @Param        id   path  int                true  "退款单ID"
// @Param        req  body  contract.AuditReq  true  "审批请求"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /refund/audit/{id} [post]
func (h *Controller) RefundAudit(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req contract.AuditReq
	if err := bind.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	if req.Approved == nil {
		response.Fail(c, errs.BadRequest(errs.ErrApproveConclusionRequired))
		return
	}
	if err := h.Svc.AuditRefund(c.Request.Context(), op, id, *req.Approved, req.Remark); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

// -------- 订单加项 --------
// 加项金额计入订单总额与尾款，增删改均由 service 在同一事务内重算订单金额并写日志。

// @Summary      订单加项列表
// @Tags         订单加项
// @Accept       json
// @Produce      json
// @Param        order_id  path  int  true  "订单ID"
// @Success      200  {object}  response.Body{data=[]model.OrderAddon}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /order/addon/list/{order_id} [post]
func (h *Controller) OrderAddonList(c *gin.Context) {
	op := middleware.GetOperator(c)
	orderID, err := bind.PathID(c, "order_id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	list, err := h.Svc.ListOrderAddons(c.Request.Context(), op, orderID)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, list)
}

// @Summary      新建订单加项
// @Description  新增加项并在同一事务内重算订单金额；body.order_id 为订单主键。
// @Tags         订单加项
// @Accept       json
// @Produce      json
// @Param        req  body  contract.OrderAddonCreateReq  true  "加项信息（含 order_id）"
// @Success      200  {object}  response.Body{data=model.OrderAddon}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /order/addon/create [post]
func (h *Controller) OrderAddonCreate(c *gin.Context) {
	op := middleware.GetOperator(c)
	orderID, err := bind.BodyID(c, "order_id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req contract.OrderAddonReq
	if err := bind.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	a, err := h.Svc.CreateOrderAddon(c.Request.Context(), op, orderID, req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, a)
}

// @Summary      更新订单加项
// @Description  更新加项并在同一事务内重算订单金额；body.id 为加项主键。
// @Tags         订单加项
// @Accept       json
// @Produce      json
// @Param        req  body  contract.OrderAddonUpdateReq  true  "加项信息（含 id）"
// @Success      200  {object}  response.Body{data=model.OrderAddon}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /order/addon/update [post]
func (h *Controller) OrderAddonUpdate(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := bind.BodyID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req contract.OrderAddonReq
	if err := bind.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	a, err := h.Svc.UpdateOrderAddon(c.Request.Context(), op, id, req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, a)
}

// @Summary      删除订单加项
// @Description  删除加项并在同一事务内重算订单金额。
// @Tags         订单加项
// @Accept       json
// @Produce      json
// @Param        id  path  int  true  "加项ID"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /order/addon/delete/{id} [post]
func (h *Controller) OrderAddonDelete(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.DeleteOrderAddon(c.Request.Context(), op, id); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

// -------- 改期 --------
// PC 端不直接改订单拍摄日期（会漏掉档期锁重排），统一走改期单链路。

// @Summary      订单改期单列表
// @Tags         改期
// @Accept       json
// @Produce      json
// @Param        order_id  path  int  true  "订单ID"
// @Success      200  {object}  response.Body{data=[]model.OrderReschedule}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /order/reschedule/list/{order_id} [post]
func (h *Controller) OrderRescheduleList(c *gin.Context) {
	op := middleware.GetOperator(c)
	orderID, err := bind.PathID(c, "order_id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	list, err := h.Svc.ListOrderReschedules(c.Request.Context(), op, orderID)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, list)
}

// @Summary      发起改期
// @Description  管理端代客户发起改期申请（改期统一走改期单链路，不直接改订单拍摄日期）。
// @Tags         改期
// @Accept       json
// @Produce      json
// @Param        order_id  path  int                        true  "订单ID"
// @Param        req       body  contract.RescheduleApplyReq true  "改期请求"
// @Success      200  {object}  response.Body{data=model.OrderReschedule}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /order/reschedule/apply/{order_id} [post]
func (h *Controller) OrderRescheduleApply(c *gin.Context) {
	op := middleware.GetOperator(c)
	orderID, err := bind.PathID(c, "order_id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req contract.RescheduleApplyReq
	if err := bind.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	r, err := h.Svc.ApplyOrderReschedule(c.Request.Context(), op, orderID, req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, r)
}

// OrderRescheduleAudit 改期审批，body: {approved, remark}（同样用 *bool，避免驳回被误判未传参）
// @Summary      改期审批
// @Description  审批改期申请；approved 为 true 通过、false 驳回，缺省返回参数错误。
// @Tags         改期
// @Accept       json
// @Produce      json
// @Param        id   path  int                true  "改期单ID"
// @Param        req  body  contract.AuditReq  true  "审批请求"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /order/reschedule/audit/{id} [post]
func (h *Controller) OrderRescheduleAudit(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req contract.AuditReq
	if err := bind.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	if req.Approved == nil {
		response.Fail(c, errs.BadRequest(errs.ErrApproveConclusionRequired))
		return
	}
	if err := h.Svc.StaffRescheduleAudit(c.Request.Context(), op, id, *req.Approved, req.Remark); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}
