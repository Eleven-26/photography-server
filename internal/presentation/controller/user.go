package controller

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"photography-server/internal/middleware"
	"photography-server/internal/response"
	"photography-server/internal/service"
)

func (h *Controller) UserList(c *gin.Context) {
	op := middleware.GetOperator(c)
	page, pageSize := pager(c)
	storeID, _ := strconv.ParseInt(c.Query("store_id"), 10, 64)
	list, total, err := h.Svc.ListUsers(op, page, pageSize, queryStr(c, "keyword"), storeID)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.PageOK(c, list, total, page, pageSize)
}

func (h *Controller) UserCreate(c *gin.Context) {
	op := middleware.GetOperator(c)
	var req service.UserCreateReq
	if err := h.bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.CreateUser(op, req); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

func (h *Controller) UserUpdate(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := pathID(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req service.UserUpdateReq
	if err := h.bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.UpdateUser(op, id, req); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

func (h *Controller) UserDelete(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := pathID(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.DeleteUser(op, id); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

func (h *Controller) UserResetPassword(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := pathID(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req struct {
		Password string `json:"password" binding:"required"`
	}
	if err := h.bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.ResetPassword(op, id, req.Password); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

func (h *Controller) RoleList(c *gin.Context) {
	op := middleware.GetOperator(c)
	list, err := h.Svc.ListRoles(op)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, list)
}

func (h *Controller) RoleCreate(c *gin.Context) {
	op := middleware.GetOperator(c)
	var req struct {
		Name   string `json:"name" binding:"required"`
		Code   string `json:"code" binding:"required"`
		Remark string `json:"remark"`
	}
	if err := h.bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.CreateRole(op, req.Name, req.Code, req.Remark); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

func (h *Controller) RoleUpdate(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := pathID(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req struct {
		Name   string `json:"name"`
		Code   string `json:"code"`
		Remark string `json:"remark"`
		Status int    `json:"status"`
	}
	if err := h.bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.UpdateRole(op, id, req.Name, req.Code, req.Remark, req.Status); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

func (h *Controller) RoleDelete(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := pathID(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.DeleteRole(op, id); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

func (h *Controller) StoreList(c *gin.Context) {
	op := middleware.GetOperator(c)
	list, err := h.Svc.ListStores(op)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, list)
}

func (h *Controller) StoreCreate(c *gin.Context) {
	op := middleware.GetOperator(c)
	var req struct {
		Name    string `json:"name" binding:"required"`
		Address string `json:"address"`
		Phone   string `json:"phone"`
	}
	if err := h.bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.CreateStore(op, req.Name, req.Address, req.Phone); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

func (h *Controller) StoreUpdate(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := pathID(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req struct {
		Name    string `json:"name"`
		Address string `json:"address"`
		Phone   string `json:"phone"`
		Status  int    `json:"status"`
	}
	if err := h.bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.UpdateStore(op, id, req.Name, req.Address, req.Phone, req.Status); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

func (h *Controller) StoreDelete(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := pathID(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.DeleteStore(op, id); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}
