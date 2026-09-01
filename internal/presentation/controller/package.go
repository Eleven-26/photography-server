package controller

import (
	"github.com/gin-gonic/gin"

	"photography-server/internal/middleware"
	"photography-server/internal/presentation/dto"
	"photography-server/internal/response"
)

func (h *Controller) PackageList(c *gin.Context) {
	op := middleware.GetOperator(c)
	page, pageSize := pager(c)
	list, total, err := h.Svc.ListPackages(op, page, pageSize,
		queryStr(c, "keyword"), queryStr(c, "status"), queryStr(c, "category"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.PageOK(c, list, total, page, pageSize)
}

func (h *Controller) PackageDetail(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := pathID(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	pkg, err := h.Svc.GetPackage(op, id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, pkg)
}

func (h *Controller) PackageCreate(c *gin.Context) {
	op := middleware.GetOperator(c)
	var req dto.PackageReq
	if err := h.bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	pkg, err := h.Svc.CreatePackage(op, req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, pkg)
}

func (h *Controller) PackageUpdate(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := pathID(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req dto.PackageReq
	if err := h.bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.UpdatePackage(op, id, req); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

func (h *Controller) PackageStatus(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := pathID(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := h.bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	_ = op
	_ = id
	_ = req
	response.OKNil(c)
}

func (h *Controller) PackageDelete(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := pathID(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.DeletePackage(op, id); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}
