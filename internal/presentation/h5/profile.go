// 客户资料（由 h5.go 按业务域拆分，2026-09-21 结构整改）。
package h5

import (
	"photography-server/internal/contract"
	"photography-server/internal/middleware"
	"photography-server/internal/pkg/response"
	"photography-server/internal/presentation/bind"

	"github.com/gin-gonic/gin"
)

// CustomerProfile 我的资料（客户中心 → 个人信息）
// @Summary      我的资料
// @Description  客户中心 → 个人信息。
// @Tags         客户区·我的
// @Accept       json
// @Produce      json
// @Success      200  {object}  response.Body{data=contract.ClientProfileResp}
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /h5/customer/profile [post]
func (h *Controller) CustomerProfile(c *gin.Context) {
	cu := middleware.GetClientUser(c)
	p, err := h.Svc.ClientProfile(c.Request.Context(), cu)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, p)
}

// CustomerProfileUpdate 客户自助修改资料。
// 这里刻意用**严格** BindJSON（而非别处可选 body 的 `_ = c.ShouldBindJSON`）：
// 请求体畸形时必须报错——若静默当成「没有字段要改」，会返回成功但什么都没保存，
// 客户端显示「已保存」而库里没变，正是最难排查的一类假故障（同封面保存那次的教训）。
// @Summary      修改我的资料
// @Description  客户自助修改资料，**只接受字段白名单**（remark / tags / level / source / status 属工作室内部信息，不可改）。
// @Description  请求体畸形时直接报错，不静默当「无字段要改」——避免返回成功但库里没变。
// @Tags         客户区·我的
// @Accept       json
// @Produce      json
// @Param        req  body  contract.ClientProfileUpdateReq  true  "资料字段（白名单）"
// @Success      200  {object}  response.Body{data=contract.ClientProfileResp}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /h5/customer/profile/update [post]
func (h *Controller) CustomerProfileUpdate(c *gin.Context) {
	cu := middleware.GetClientUser(c)
	var req contract.ClientProfileUpdateReq
	if err := bind.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	p, err := h.Svc.ClientUpdateProfile(c.Request.Context(), cu, req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, p)
}

// PhotographerOptions 定制需求页「选择门店 → 选择摄影师」的候选（客户中心 → 定制需求）。
//
// 候选 = 该客户**曾下过单或提过定制需求**的门店与摄影师，外加本次分享链接带入的分享人
// （见 service.ClientPhotographerOptions）。无候选（新客户且非分享进入）时返回空数组，
// 前端不展示选择器，需求仍可提交、由工作室后续指派。
// @Summary      门店/摄影师候选
// @Description  定制需求页「选择门店 → 选择摄影师」的候选 = 该客户曾下过单或提过需求的门店与摄影师，外加分享链接的分享人。
// @Description  无候选（新客户且非分享进入）时返回空数组，前端不展示选择器，需求仍可提交。
// @Tags         客户区·我的
// @Accept       json
// @Produce      json
// @Success      200  {object}  response.Body{data=[]contract.ClientStoreOption}
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /h5/customer/photographer-options [post]
func (h *Controller) PhotographerOptions(c *gin.Context) {
	cu := middleware.GetClientUser(c)
	opts, err := h.Svc.ClientPhotographerOptions(c.Request.Context(), cu, staffFrom(c))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, opts)
}
