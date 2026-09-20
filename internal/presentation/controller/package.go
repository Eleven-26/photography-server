package controller

import (
	"github.com/gin-gonic/gin"

	"photography-server/internal/contract"
	"photography-server/internal/enum"
	"photography-server/internal/middleware"
	"photography-server/internal/pkg/response"
	"photography-server/internal/presentation/bind"
)

// @Summary      套餐列表
// @Description  分页查询套餐，支持关键词、上下架状态、分类过滤。
// @Tags         套餐
// @Accept       json
// @Produce      json
// @Param        req  body  object{page=int,page_size=int,keyword=string,status=string,category=string}  true  "查询条件（status 1-草稿 2-已上架 3-已下线）"
// @Success      200  {object}  response.Body{data=response.Page{list=[]model.Package,total=int,page=int,page_size=int}}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /package/list [post]
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

// @Summary      套餐详情
// @Description  返回套餐详情。
// @Tags         套餐
// @Accept       json
// @Produce      json
// @Param        id  path  int  true  "套餐ID"
// @Success      200  {object}  response.Body{data=model.Package}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /package/detail/{id} [post]
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

// @Summary      创建套餐
// @Description  创建套餐。套餐是公司（slug）维度资源，不归属单个摄影师。
// @Tags         套餐
// @Accept       json
// @Produce      json
// @Param        req  body  contract.PackageReq  true  "套餐信息"
// @Success      200  {object}  response.Body{data=model.Package}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /package/create [post]
func (h *Controller) PackageCreate(c *gin.Context) {
	op := middleware.GetOperator(c)
	var req contract.PackageReq
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

// @Summary      更新套餐
// @Description  更新套餐；body.id 为主键。**在售套餐改价会被守卫拦截**（ErrPackageActiveUpdate），避免在售套餐被静默改价。
// @Tags         套餐
// @Accept       json
// @Produce      json
// @Param        req  body  contract.PackageReq  true  "套餐信息（含 id）"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /package/update [post]
func (h *Controller) PackageUpdate(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := bind.BodyID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req contract.PackageReq
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
// @Summary      套餐上下架
// @Description  切换套餐上架 / 下线状态。
// @Tags         套餐
// @Accept       json
// @Produce      json
// @Param        id   path  int                  true  "套餐ID"
// @Param        req  body  object{status=int}   true  "status 1-草稿 2-已上架 3-已下线（enum.PackageStatus，数字类型）"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /package/status/{id} [post]
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

// @Summary      删除套餐
// @Description  删除套餐。**在售套餐会被守卫拦截**（ErrPackageActiveDelete）。
// @Tags         套餐
// @Accept       json
// @Produce      json
// @Param        id  path  int  true  "套餐ID"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /package/delete/{id} [post]
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
