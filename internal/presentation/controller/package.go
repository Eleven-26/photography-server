package controller

import (
	"github.com/gin-gonic/gin"

	"photography-server/internal/enum"
	"photography-server/internal/middleware"
	"photography-server/internal/presentation/bind"
	"photography-server/internal/presentation/dto"
	"photography-server/internal/presentation/response"
)

func (h *Controller) PackageList(c *gin.Context) {
	op := middleware.GetOperator(c)
	page, pageSize := bind.Pager(c)
	list, total, err := h.Svc.ListPackages(c.Request.Context(), op, page, pageSize,
		bind.ParamStr(c, "keyword"), bind.ParamStr(c, "status"), bind.ParamStr(c, "category"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.PageOK(c, list, total, page, pageSize)
}

func (h *Controller) PackageDetail(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	pkg, err := h.Svc.GetPackage(c.Request.Context(), op, id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, pkg)
}

func (h *Controller) PackageCreate(c *gin.Context) {
	op := middleware.GetOperator(c)
	var req dto.PackageReq
	if err := bind.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	pkg, err := h.Svc.CreatePackage(c.Request.Context(), op, req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, pkg)
}

func (h *Controller) PackageUpdate(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req dto.PackageReq
	if err := bind.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.UpdatePackage(c.Request.Context(), op, id, req); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

// PackageStatus 套餐上架/下线（PC 卡片开关）。
// 注意：前端 `setPackageStatus(id, status: number)` 传的是数字（enum.PackageStatus），
// 原实现声明 `Status string` + `binding:"required"`，JSON 反序列化会直接抛
// "cannot unmarshal number into Go struct field .status of type string"，即用户看到的类型错误。
func (h *Controller) PackageStatus(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req struct {
		Status int `json:"status" binding:"required"`
	}
	if err := bind.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.ChangePackageStatus(c.Request.Context(), op, id, enum.PackageStatus(req.Status)); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

func (h *Controller) PackageDelete(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.DeletePackage(c.Request.Context(), op, id); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}
