// 作品集（由 h5.go 按业务域拆分，2026-09-21 结构整改）。
package h5

import (
	"photography-server/internal/pkg/params"
	"photography-server/internal/pkg/response"
	"photography-server/internal/presentation/bind"

	"github.com/gin-gonic/gin"
)

// AssetList 公开作品列表（预约主页作品集）
// @Summary      公开作品列表
// @Description  预约主页作品集，**只出「已发布 + 公开」的作品**。
// @Description  租户定位：body.slug（也可放 query slug / X-Slug 头）；服务端按 slug 反查 company_id，**不接受客户端直传 company_id**。
// @Tags         客户区·作品集
// @Accept       json
// @Produce      json
// @Param        req  body  object{page=int,page_size=int,category=string,featured=string}  true  "查询条件（featured=1 只看精选；slug 见说明）"
// @Success      200  {object}  response.Body{data=response.Page{list=[]model.Asset,total=int,page=int,page_size=int}}
// @Failure      400  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Router       /h5/asset/list [post]
func (h *Controller) AssetList(c *gin.Context) {
	companyID, err := h.requireCompany(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	page, pageSize := bind.Pager(c)
	list, total, err := h.Svc.ClientAssets(c.Request.Context(), companyID, page, pageSize,
		params.Str(c, "category"), params.Str(c, "featured") == "1")
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.PageOK(c, list, total, page, pageSize)
}

// AssetDetail 公开作品详情（浏览数 +1）
// @Summary      公开作品详情
// @Description  作品详情，附带浏览数 +1。
// @Description  租户定位：body.slug（也可放 query slug / X-Slug 头）；服务端按 slug 反查 company_id，**不接受客户端直传 company_id**。
// @Tags         客户区·作品集
// @Accept       json
// @Produce      json
// @Param        id  path  int  true  "作品ID"
// @Success      200  {object}  response.Body{data=model.Asset}
// @Failure      400  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Router       /h5/asset/detail/{id} [post]
func (h *Controller) AssetDetail(c *gin.Context) {
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
	a, err := h.Svc.ClientAssetDetail(c.Request.Context(), companyID, id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, a)
}
