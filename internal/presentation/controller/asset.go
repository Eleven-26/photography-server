package controller

import (
	"github.com/gin-gonic/gin"

	"photography-server/internal/middleware"
	"photography-server/internal/presentation/bind"
	"photography-server/internal/presentation/dto"
	"photography-server/internal/presentation/response"
)

func (h *Controller) AssetList(c *gin.Context) {
	op := middleware.GetOperator(c)
	page, pageSize := bind.Pager(c)
	list, total, err := h.Svc.ListAssets(c.Request.Context(), op, page, pageSize,
		bind.ParamStr(c, "keyword"), bind.ParamStr(c, "category"), bind.ParamStr(c, "status"), bind.ParamStr(c, "featured"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.PageOK(c, list, total, page, pageSize)
}

func (h *Controller) AssetDetail(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	a, err := h.Svc.GetAsset(c.Request.Context(), op, id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, a)
}

func (h *Controller) AssetCreate(c *gin.Context) {
	op := middleware.GetOperator(c)
	var req dto.AssetCreateReq
	if err := bind.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	a, err := h.Svc.CreateAsset(c.Request.Context(), op, req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, a)
}

func (h *Controller) AssetUpdate(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req dto.AssetUpdateReq
	if err := bind.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.UpdateAsset(c.Request.Context(), op, id, req); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

func (h *Controller) AssetDelete(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.DeleteAsset(c.Request.Context(), op, id); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

// AssetStatus 作品「发布状态 / 可见性 / 精选」开关（局部更新，不要求回传全字段）
func (h *Controller) AssetStatus(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req dto.AssetFlagsReq
	if err := bind.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.UpdateAssetFlags(c.Request.Context(), op, id, req); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}
