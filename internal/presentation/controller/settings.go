package controller

import (
	"github.com/gin-gonic/gin"

	"photography-server/internal/contract"
	"photography-server/internal/middleware"
	"photography-server/internal/pkg/response"
	"photography-server/internal/presentation/bind"
)

func (h *Controller) Workspace(c *gin.Context) {
	op := middleware.GetOperator(c)
	w, err := h.Svc.Workspace(c.Request.Context(), op)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, w)
}

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

func (h *Controller) PaymentMethodList(c *gin.Context) {
	op := middleware.GetOperator(c)
	list, err := h.Svc.ListPaymentMethods(c.Request.Context(), op)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, list)
}

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
