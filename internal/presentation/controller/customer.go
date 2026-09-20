package controller

import (
	"github.com/gin-gonic/gin"

	"photography-server/internal/contract"
	"photography-server/internal/middleware"
	"photography-server/internal/pkg/response"
	"photography-server/internal/presentation/bind"
)

// CustomerList 客户列表
//
// @Summary      客户列表
// @Description  分页查询客户，支持 keyword 模糊匹配姓名/手机号。
// @Tags         客户
// @Accept       json
// @Produce      json
// @Param        req  body  contract.CustomerListReq  true  "查询条件"
// @Success      200  {object}  response.Body{data=response.Page{list=[]model.Customer,total=int,page=int,page_size=int}}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      403  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /customer/list [post]
func (h *Controller) CustomerList(c *gin.Context) {
	op := middleware.GetOperator(c)
	page, pageSize := bind.Pager(c)
	list, total, err := h.Svc.ListCustomers(c.Request.Context(), op, page, pageSize, bind.ParamStr(c, "keyword"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.PageOK(c, list, total, page, pageSize)
}

// @Summary      客户详情
// @Description  按 ID 查询客户档案。
// @Tags         客户
// @Accept       json
// @Produce      json
// @Param        id  path  int  true  "客户ID"
// @Success      200  {object}  response.Body{data=model.Customer}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /customer/detail/{id} [post]
func (h *Controller) CustomerDetail(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	detail, err := h.Svc.GetCustomer(c.Request.Context(), op, id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, detail)
}

// @Summary      新建客户
// @Description  创建客户档案；手机号在租户内唯一，重复会返回冲突。
// @Tags         客户
// @Accept       json
// @Produce      json
// @Param        req  body  contract.CustomerCreateReq  true  "客户信息"
// @Success      200  {object}  response.Body{data=model.Customer}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      409  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /customer/create [post]
func (h *Controller) CustomerCreate(c *gin.Context) {
	op := middleware.GetOperator(c)
	var req contract.CustomerCreateReq
	if err := bind.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	customer, err := h.Svc.CreateCustomer(c.Request.Context(), op, req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, customer)
}

// @Summary      更新客户
// @Description  更新客户档案；body.id 为主键。
// @Tags         客户
// @Accept       json
// @Produce      json
// @Param        req  body  contract.CustomerUpdateReq  true  "客户信息（含 id）"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /customer/update [post]
func (h *Controller) CustomerUpdate(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := bind.BodyID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req contract.CustomerUpdateReq
	if err := bind.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.UpdateCustomer(c.Request.Context(), op, id, req); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

// @Summary      删除客户
// @Tags         客户
// @Accept       json
// @Produce      json
// @Param        id  path  int  true  "客户ID"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /customer/delete/{id} [post]
func (h *Controller) CustomerDelete(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.DeleteCustomer(c.Request.Context(), op, id); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

// @Summary      客户统计
// @Description  客户总量、状态分布、复购等看板指标。
// @Tags         客户
// @Accept       json
// @Produce      json
// @Success      200  {object}  response.Body{data=contract.CustomerStatsResp}
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /customer/stats [post]
func (h *Controller) CustomerStats(c *gin.Context) {
	op := middleware.GetOperator(c)
	st, err := h.Svc.GetCustomerStats(c.Request.Context(), op)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, st)
}

// CustomerOrders 客户名下订单
// @Summary      客户名下订单
// @Description  分页返回指定客户名下的订单列表。
// @Tags         客户
// @Accept       json
// @Produce      json
// @Param        id   path  int               true  "客户ID"
// @Param        req  body  contract.PageReq  true  "分页参数"
// @Success      200  {object}  response.Body{data=response.Page{list=[]model.Order,total=int,page=int,page_size=int}}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /customer/orders/{id} [post]
func (h *Controller) CustomerOrders(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	page, pageSize := bind.Pager(c)
	list, total, err := h.Svc.ListOrders(c.Request.Context(), op, page, pageSize, "", id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.PageOK(c, list, total, page, pageSize)
}
