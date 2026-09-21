// 订单 / 收款 / 退款 / 改期审核（由 staff.go 按业务域拆分，2026-09-21 结构整改）。
package wechat

import (
	"photography-server/internal/contract"
	"photography-server/internal/middleware"
	"photography-server/internal/pkg/params"
	"photography-server/internal/pkg/response"
	"photography-server/internal/presentation/bind"

	"github.com/gin-gonic/gin"
)

// RescheduleList 改期单列表
// @Summary      改期单列表
// @Description  分页查询改期单，可按状态过滤。（发起改期复用 PC 的 /order/reschedule/apply/:order_id）
// @Tags         员工端·改期
// @Accept       json
// @Produce      json
// @Param        req  body  object{page=int,page_size=int,status=int}  true  "查询条件（status 0=全部）"
// @Success      200  {object}  response.Body{data=response.Page{list=[]model.OrderReschedule,total=int,page=int,page_size=int}}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /wechat/staff/reschedule/list [post]
func (h *Controller) RescheduleList(c *gin.Context) {
	op := middleware.GetOperator(c)
	page, pageSize := bind.Pager(c)
	status := params.Int(c, "status")
	list, total, err := h.Svc.StaffRescheduleList(c.Request.Context(), op, page, pageSize, status)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.PageOK(c, list, total, page, pageSize)
}

// RescheduleAudit 改期审批
// @Summary      改期审批
// @Tags         员工端·改期
// @Accept       json
// @Produce      json
// @Param        id   path  int                                 true  "改期单ID"
// @Param        req  body  contract.StaffRescheduleAuditReq    true  "审批结论"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /wechat/staff/reschedule/audit/{id} [post]
func (h *Controller) RescheduleAudit(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req contract.StaffRescheduleAuditReq
	if err := bind.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.StaffRescheduleAudit(c.Request.Context(), op, id, req.Approved, req.Remark); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

// RefundAudit 退款审核
// @Summary      退款审核
// @Description  **端差异实现**：员工端契约是 {"approve": bool}，PC 是 {"approved": *bool} 且必填；改任一侧都是破坏性变更。
// @Description  approve=false 表示驳回，布尔零值合法，故不使用 required 校验。
// @Tags         员工端·退款
// @Accept       json
// @Produce      json
// @Param        id   path  int                              true  "退款单ID"
// @Param        req  body  object{approve=bool,remark=string} true  "审核结论（approve=false 即驳回）"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /wechat/staff/refund/audit/{id} [post]
func (h *Controller) RefundAudit(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req struct {
		Approve bool   `json:"approve"` // 是否通过（false=驳回，不使用 required 以放行布尔零值）
		Remark  string `json:"remark"`  // 审核备注
	}
	if err := bind.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.AuditRefund(c.Request.Context(), op, id, req.Approve, req.Remark); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}
