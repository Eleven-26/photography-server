package controller

import (
	"github.com/gin-gonic/gin"

	"photography-server/internal/contract"
	"photography-server/internal/middleware"
	"photography-server/internal/pkg/response"
	"photography-server/internal/presentation/bind"
)

// @Summary      工作台信息
// @Description  返回当前操作人所在租户的工作台聚合信息（公司 / 门店 / 角色 / 权限点）。
// @Tags         工作室设置
// @Accept       json
// @Produce      json
// @Success      200  {object}  response.Body{data=contract.WorkspaceResp}
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /settings/workspace [post]
func (h *Controller) Workspace(c *gin.Context) {
	op := middleware.GetOperator(c)
	w, err := h.Svc.Workspace(c.Request.Context(), op)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, w)
}

// @Summary      更新公司信息
// @Description  更新租户公司资料。
// @Tags         工作室设置
// @Accept       json
// @Produce      json
// @Param        req  body  contract.CompanyUpdateReq  true  "公司信息"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /settings/company/update [post]
func (h *Controller) CompanyUpdate(c *gin.Context) {
	op := middleware.GetOperator(c)
	var req contract.CompanyUpdateReq
	if err := bind.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.UpdateCompany(c.Request.Context(), op, req); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

// @Summary      收款方式列表
// @Description  返回工作室收款方式（资金不经平台，仅登记）。与员工端 /wechat/staff/settings/payment-method/list 复用同一实现。
// @Tags         工作室设置
// @Accept       json
// @Produce      json
// @Success      200  {object}  response.Body{data=[]model.PaymentMethod}
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /settings/payment-method/list [post]
func (h *Controller) PaymentMethodList(c *gin.Context) {
	op := middleware.GetOperator(c)
	list, err := h.Svc.ListPaymentMethods(c.Request.Context(), op)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, list)
}

// @Summary      新建收款方式
// @Tags         工作室设置
// @Accept       json
// @Produce      json
// @Param        req  body  contract.PaymentMethodReq  true  "收款方式信息"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /settings/payment-method/create [post]
func (h *Controller) PaymentMethodCreate(c *gin.Context) {
	op := middleware.GetOperator(c)
	var req contract.PaymentMethodReq
	if err := bind.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.CreatePaymentMethod(c.Request.Context(), op, req); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

// @Summary      更新收款方式
// @Description  更新收款方式；body.id 为主键。
// @Tags         工作室设置
// @Accept       json
// @Produce      json
// @Param        req  body  contract.PaymentMethodReq  true  "收款方式信息（含 id）"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /settings/payment-method/update [post]
func (h *Controller) PaymentMethodUpdate(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := bind.BodyID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req contract.PaymentMethodReq
	if err := bind.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.UpdatePaymentMethod(c.Request.Context(), op, id, req); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

// @Summary      删除收款方式
// @Tags         工作室设置
// @Accept       json
// @Produce      json
// @Param        id  path  int  true  "收款方式ID"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /settings/payment-method/delete/{id} [post]
func (h *Controller) PaymentMethodDelete(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.DeletePaymentMethod(c.Request.Context(), op, id); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

// @Summary      操作日志列表
// @Description  分页查询操作日志，支持关键词、模块、状态过滤。
// @Tags         工作室设置
// @Accept       json
// @Produce      json
// @Param        req  body  object{page=int,page_size=int,keyword=string,module=string,status=string}  true  "查询条件"
// @Success      200  {object}  response.Body{data=response.Page{list=[]model.SysOperationLog,total=int,page=int,page_size=int}}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /settings/operation-log/list [post]
func (h *Controller) OperationLogList(c *gin.Context) {
	op := middleware.GetOperator(c)
	page, pageSize := bind.Pager(c)
	list, total, err := h.Svc.ListOperationLogs(c.Request.Context(), op, page, pageSize,
		bind.ParamStr(c, "keyword"), bind.ParamStr(c, "module"), bind.ParamStr(c, "status"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.PageOK(c, list, total, page, pageSize)
}

// ---------------------------------------------------------------------
// 工作室设置（预约主页 / 接单规则 / 改期政策）—— 与员工端共用 service
// ---------------------------------------------------------------------

// StudioGet 工作室设置读取（不存在时自动建默认行）。
// 响应在 model.StudioSetting 之上追加 homepage_url / portfolio_url
// （服务端按 share.h5_base_url 拼装），供前端直接展示与复制，前端不再自行拼域名。
// 链接另行拼入分享人账号 ID（&staff_id=op.UserID）：客户从谁的链接下单，订单就归到谁名下。
// @Summary      读取工作室设置
// @Description  读取工作室设置（预约主页 / 接单规则 / 改期政策）；不存在时自动建默认行。
// @Description  响应在 model.StudioSetting 之上追加 homepage_url / portfolio_url 两条分享链接（域名由服务端下发，
// @Description  并已拼入分享人账号 ID &staff_id=<当前操作人>），前端直接展示与复制，不再自行拼域名。与员工端复用同一实现。
// @Tags         工作室设置
// @Accept       json
// @Produce      json
// @Success      200  {object}  response.Body{data=contract.StaffStudioSettingResp}
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /settings/studio/get [post]
func (h *Controller) StudioGet(c *gin.Context) {
	op := middleware.GetOperator(c)
	st, err := h.Svc.StudioSetting(c.Request.Context(), op)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, contract.NewStaffStudioSettingResp(st, h.Cfg.Share.H5BaseURL, op.UserID))
}

// StudioUpdate 工作室设置更新（仅更新传入字段，数值支持改为 0）
// @Summary      更新工作室设置
// @Description  更新工作室设置，**仅更新传入字段**（数值支持改为 0）。与员工端复用同一实现。
// @Tags         工作室设置
// @Accept       json
// @Produce      json
// @Param        req  body  contract.StaffStudioSettingReq  true  "设置字段（未传字段保持原值）"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /settings/studio/update [post]
func (h *Controller) StudioUpdate(c *gin.Context) {
	op := middleware.GetOperator(c)
	var req contract.StaffStudioSettingReq
	if err := bind.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.UpdateStudioSetting(c.Request.Context(), op, req.ToUpdates()); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}
