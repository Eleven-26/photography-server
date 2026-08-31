package controller

import (
	"github.com/gin-gonic/gin"

	"photography-server/internal/middleware"
	"photography-server/internal/response"
	"photography-server/internal/service"
)

func (h *Controller) AssetList(c *gin.Context) {
	op := middleware.GetOperator(c)
	page, pageSize := pager(c)
	list, total, err := h.Svc.ListAssets(op, page, pageSize,
		queryStr(c, "keyword"), queryStr(c, "category"), queryStr(c, "status"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.PageOK(c, list, total, page, pageSize)
}

func (h *Controller) AssetDetail(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := pathID(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	a, err := h.Svc.GetAsset(op, id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, a)
}

func (h *Controller) AssetCreate(c *gin.Context) {
	op := middleware.GetOperator(c)
	var req service.AssetReq
	if err := h.bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	a, err := h.Svc.CreateAsset(op, req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, a)
}

func (h *Controller) AssetUpdate(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := pathID(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req service.AssetReq
	if err := h.bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.UpdateAsset(op, id, req); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

func (h *Controller) AssetDelete(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := pathID(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.DeleteAsset(op, id); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}
