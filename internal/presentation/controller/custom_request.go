package controller

import (
	"github.com/gin-gonic/gin"

	"photography-server/internal/contract"
	"photography-server/internal/middleware"
	"photography-server/internal/pkg/response"
	"photography-server/internal/presentation/bind"
)

// CustomRequestList 定制需求列表（管理端）。
// query: status 1-待处理 2-已响应 3-已关闭；0 或不传 = 全部。
// query: photographer_id 按「客户指定的摄影师」筛选；0 或不传 = 不过滤。
// 与员工端 /wechat/staff/custom-request/list 是同一 Handler（endpoints.go 复用本实现），
// 两端返回结构一致：{list,total,page,page_size}。
// @Summary      定制需求列表
// @Description  分页查询定制需求。与员工端 /wechat/staff/custom-request/list 是同一 Handler，两端结构一致。
// @Tags         定制需求
// @Accept       json
// @Produce      json
// @Param        req  body  object{page=int,page_size=int,keyword=string,status=int,photographer_id=int}  true  "查询条件（status 1-待处理 2-已响应 3-已关闭，0/不传=全部；photographer_id 0/不传=不过滤）"
// @Success      200  {object}  response.Body{data=response.Page{list=[]model.CustomRequest,total=int,page=int,page_size=int}}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /custom-request/list [post]
func (h *Controller) CustomRequestList(c *gin.Context) {
	op := middleware.GetOperator(c)
	page, pageSize := bind.Pager(c)
	list, total, err := h.Svc.StaffCustomRequests(c.Request.Context(), op, page, pageSize,
		bind.ParamInt(c, "status"), bind.ParamInt64(c, "photographer_id"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.PageOK(c, list, total, page, pageSize)
}

// CustomRequestRespond 响应定制需求（:id 为定制需求 ID）body: {response}
// @Summary      响应定制需求
// @Description  填写对定制需求的响应内容，需求置为「已响应」。
// @Tags         定制需求
// @Accept       json
// @Produce      json
// @Param        id   path  int                                    true  "定制需求ID"
// @Param        req  body  contract.StaffCustomRequestRespondReq  true  "响应内容"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /custom-request/respond/{id} [post]
func (h *Controller) CustomRequestRespond(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req contract.StaffCustomRequestRespondReq
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
// @Summary      定制需求转订单
// @Description  把定制需求转为订单。⚠️ 需求**不含套餐**，必须在此指定 package_id；成功后需求自动置为「已响应」。转单复用 CreateOrder。
// @Tags         定制需求
// @Accept       json
// @Produce      json
// @Param        id   path  int                        true  "定制需求ID"
// @Param        req  body  contract.OrderCreateReq    true  "订单信息（package_id 必填）"
// @Success      200  {object}  response.Body{data=model.Order}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /custom-request/convert/{id} [post]
func (h *Controller) CustomRequestConvert(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req contract.OrderCreateReq
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
