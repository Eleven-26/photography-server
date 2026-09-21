// 套餐 / 工作室 / 可约档期浏览（由 h5.go 按业务域拆分，2026-09-21 结构整改）。
package h5

import (
	"photography-server/internal/pkg/params"
	"photography-server/internal/pkg/response"
	"photography-server/internal/presentation/bind"

	"github.com/gin-gonic/gin"
)

// PackageList 套餐列表（已上架）
// @Summary      套餐列表
// @Description  预约主页套餐列表，**只返回已上架套餐**。
// @Description  租户定位：body.slug（也可放 query slug / X-Slug 头）；服务端按 slug 反查 company_id，**不接受客户端直传 company_id**。
// @Tags         客户区·套餐
// @Accept       json
// @Produce      json
// @Param        req  body  object{page=int,page_size=int,category=string}  true  "查询条件（slug 见说明）"
// @Success      200  {object}  response.Body{data=response.Page{list=[]model.Package,total=int,page=int,page_size=int}}
// @Failure      400  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Router       /h5/package/list [post]
func (h *Controller) PackageList(c *gin.Context) {
	companyID, err := h.requireCompany(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	page, pageSize := bind.Pager(c)
	list, total, err := h.Svc.ClientPackages(c.Request.Context(), companyID, page, pageSize, params.Str(c, "category"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.PageOK(c, list, total, page, pageSize)
}

// PackageDetail 套餐详情
// @Summary      套餐详情
// @Description  套餐详情（预约主页点开套餐卡片）。
// @Description  租户定位：body.slug（也可放 query slug / X-Slug 头）；服务端按 slug 反查 company_id，**不接受客户端直传 company_id**。
// @Tags         客户区·套餐
// @Accept       json
// @Produce      json
// @Param        id  path  int  true  "套餐ID"
// @Success      200  {object}  response.Body{data=model.Package}
// @Failure      400  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Router       /h5/package/detail/{id} [post]
func (h *Controller) PackageDetail(c *gin.Context) {
	companyID, err := h.requireCompany(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	pkg, err := h.Svc.ClientPackageDetail(c.Request.Context(), companyID, id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, pkg)
}

// StudioInfo 工作室预约主页信息
// @Summary      工作室信息
// @Description  预约主页展示用的工作室信息（名称 / 简介 / 联系方式 / 分享链接等）。
// @Description  租户定位：body.slug（也可放 query slug / X-Slug 头）；服务端按 slug 反查 company_id，**不接受客户端直传 company_id**。
// @Tags         客户区·工作室
// @Accept       json
// @Produce      json
// @Success      200  {object}  response.Body{data=model.StudioSetting}
// @Failure      400  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Router       /h5/studio/info [post]
func (h *Controller) StudioInfo(c *gin.Context) {
	companyID, err := h.requireCompany(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	info, err := h.Svc.ClientStudioInfo(c.Request.Context(), companyID)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, info)
}

// SlotList 指定日期可约时段（body: date、photographer_id 可选）
// @Summary      可约时段
// @Description  查询指定日期的可约时段；photographer_id 可选，用于只看某位摄影师的空闲时段。
// @Description  租户定位：body.slug（也可放 query slug / X-Slug 头）；服务端按 slug 反查 company_id，**不接受客户端直传 company_id**。
// @Tags         客户区·工作室
// @Accept       json
// @Produce      json
// @Param        req  body  object{date=string,photographer_id=int}  true  "查询条件（date 格式 2006-01-02；slug 见说明）"
// @Success      200  {object}  response.Body{data=[]contract.ClientSlot}
// @Failure      400  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Router       /h5/slot/list [post]
func (h *Controller) SlotList(c *gin.Context) {
	companyID, err := h.requireCompany(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	photographerID := params.Int64(c, "photographer_id")
	slots, err := h.Svc.ClientSlots(c.Request.Context(), companyID, params.Str(c, "date"), photographerID)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, slots)
}
