package controller

import (
	"github.com/gin-gonic/gin"

	"photography-server/internal/contract"
	"photography-server/internal/middleware"
	"photography-server/internal/pkg/response"
	"photography-server/internal/presentation/bind"
)

// @Summary      作品列表
// @Description  分页查询作品集，支持关键词、分类、发布状态、精选过滤。
// @Tags         作品集
// @Accept       json
// @Produce      json
// @Param        req  body  object{page=int,page_size=int,keyword=string,category=string,status=string,featured=string}  true  "查询条件（status 1-草稿 2-已发布；featured 0-否 1-是）"
// @Success      200  {object}  response.Body{data=response.Page{list=[]model.Asset,total=int,page=int,page_size=int}}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /asset/list [post]
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

// @Summary      作品详情
// @Description  返回单条作品详情。
// @Tags         作品集
// @Accept       json
// @Produce      json
// @Param        id  path  int  true  "作品ID"
// @Success      200  {object}  response.Body{data=model.Asset}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /asset/detail/{id} [post]
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

// @Summary      创建作品
// @Description  创建作品集条目；images 与 package_ids 为逗号分隔字符串。
// @Tags         作品集
// @Accept       json
// @Produce      json
// @Param        req  body  contract.AssetCreateReq  true  "作品信息（title 必填）"
// @Success      200  {object}  response.Body{data=model.Asset}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /asset/create [post]
func (h *Controller) AssetCreate(c *gin.Context) {
	op := middleware.GetOperator(c)
	var req contract.AssetCreateReq
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

// @Summary      更新作品
// @Description  更新作品；body.id 为主键。状态/可见性/精选/授权传 null 表示保持原值（0 是合法取值）。
// @Tags         作品集
// @Accept       json
// @Produce      json
// @Param        req  body  contract.AssetUpdateReq  true  "作品信息（含 id）"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /asset/update [post]
func (h *Controller) AssetUpdate(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := bind.BodyID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req contract.AssetUpdateReq
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

// @Summary      删除作品
// @Description  删除作品集条目。
// @Tags         作品集
// @Accept       json
// @Produce      json
// @Param        id  path  int  true  "作品ID"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /asset/delete/{id} [post]
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
// @Summary      作品状态开关
// @Description  局部切换发布状态 / 可见性 / 精选，不要求回传全字段；字段传 null 表示保持原值。
// @Tags         作品集
// @Accept       json
// @Produce      json
// @Param        id   path  int                     true  "作品ID"
// @Param        req  body  contract.AssetFlagsReq  true  "开关字段（status 1-草稿 2-已发布；visibility 1-公开 2-未公开；featured 0-否 1-是）"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /asset/status/{id} [post]
func (h *Controller) AssetStatus(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req contract.AssetFlagsReq
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
