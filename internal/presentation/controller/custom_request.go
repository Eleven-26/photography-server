package controller

import (
	"github.com/gin-gonic/gin"

	"photography-server/internal/middleware"
	"photography-server/internal/presentation/bind"
	"photography-server/internal/presentation/dto"
	"photography-server/internal/presentation/response"
)

// CustomRequestList 定制需求列表（管理端）。
// query: status 1-待处理 2-已响应 3-已关闭；0 或不传 = 全部。
// 与员工端 /wechat/staff/custom-request/list 是同一 Handler（endpoints.go 复用本实现），
// 两端返回结构一致：{list,total,page,page_size}。
func (h *Controller) CustomRequestList(c *gin.Context) {
	op := middleware.GetOperator(c)
	page, pageSize := bind.Pager(c)
	list, total, err := h.Svc.StaffCustomRequests(c.Request.Context(), op, page, pageSize, bind.ParamInt(c, "status"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.PageOK(c, list, total, page, pageSize)
}

// CustomRequestRespond 响应定制需求（:id 为定制需求 ID）body: {response}
func (h *Controller) CustomRequestRespond(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req dto.StaffCustomRequestRespondReq
	if err := bind.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.StaffCustomRequestRespond(c.Request.Context(), op, id, req.Response); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

// CustomRequestConvert 定制需求转订单（:id 为定制需求 ID）body: OrderCreateReq（package_id 必填）。
// 定制需求本身不含套餐，转单需在后台选定套餐；成功后需求自动置为「已响应」。
func (h *Controller) CustomRequestConvert(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req dto.OrderCreateReq
	if err := bind.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	o, err := h.Svc.ConvertCustomRequestToOrder(c.Request.Context(), op, id, req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, o)
}
