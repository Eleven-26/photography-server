package wechat

import (
	"github.com/gin-gonic/gin"

	"photography-server/internal/middleware"
	"photography-server/internal/pkg/params"
	"photography-server/internal/presentation/bind"
	"photography-server/internal/presentation/dto"
	"photography-server/internal/presentation/response"
)

// 小程序员工区接口（订单处理 + 日程 + 线索 AI 简报 + 个人中心）。
// 员工（摄影师/助理）通过小程序处理业务，身份经 StaffAuth 注入 Operator。
//
// 路由注册（2026-09-12 路由整理后）已迁到 router/endpoints.go 的 staffEndpoint：
// 与 PC 同路径的路由直接复用管理端 handler（同一函数指针，表在 routes 包），
// 本文件只保留【员工端独有 handler】与【真实端差异实现】（退款审核 approve / 创建交付单不收 body）。
// Controller 类型与 New 构造见 wechat.go。

// RegisterStaffPublic 注册员工区公开路由（验证码登录，挂 /wechat/staff）
func (h *Controller) RegisterStaffPublic(g *gin.RouterGroup) {
	g.POST("/auth/sms-code", h.StaffSmsCode)
	g.POST("/auth/login", h.StaffLogin)
}

// ---------------------------------------------------------------------
// 公开接口
// ---------------------------------------------------------------------

// SmsCode 发送登录验证码
func (h *Controller) StaffSmsCode(c *gin.Context) {
	var req struct {
		Mobile string `json:"mobile" binding:"required"`
	}
	if err := bind.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.SendSmsCode(c.Request.Context(), "login", req.Mobile); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

// StaffLogin 员工手机号验证码登录（按 sys_user.mobile 匹配员工）
func (h *Controller) StaffLogin(c *gin.Context) {
	var req struct {
		Mobile     string `json:"mobile" binding:"required"`
		Code       string `json:"code" binding:"required"`
		DeviceName string `json:"device_name"` // 设备名称（选填，用于设备管理）
		Platform   string `json:"platform"`    // ios/android
	}
	if err := bind.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	u, token, err := h.Svc.StaffSmsLogin(c.Request.Context(), req.Mobile, req.Code, req.DeviceName, req.Platform, c.ClientIP())
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{
		"token": token,
		"user": gin.H{
			"id":       u.ID,
			"username": u.Username,
			"nickname": u.Nickname,
			"avatar":   u.Avatar,
			"mobile":   u.Mobile,
			"role_id":  u.RoleID,
			"store_id": u.StoreID,
		},
	})
}

// ---------------------------------------------------------------------
// 工作台 / 订单
// ---------------------------------------------------------------------

// Overview 工作台待办统计
func (h *Controller) Overview(c *gin.Context) {
	op := middleware.GetOperator(c)
	ov, err := h.Svc.StaffOverview(c.Request.Context(), op)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, ov)
}

// ---------------------------------------------------------------------
// 收款 / 交付 / 改期 / 退款
// ---------------------------------------------------------------------

// DeliveryCreate 创建交付单
func (h *Controller) DeliveryCreate(c *gin.Context) {
	op := middleware.GetOperator(c)
	orderID, err := bind.PathID(c, "order_id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	d, err := h.Svc.CreateDelivery(c.Request.Context(), op, orderID)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, d)
}

// RescheduleList 改期单列表
func (h *Controller) RescheduleList(c *gin.Context) {
	op := middleware.GetOperator(c)
	page, pageSize := bind.Pager(c)
	status := params.Int(c, "status")
	list, total, err := h.Svc.StaffRescheduleList(c.Request.Context(), op, page, pageSize, status)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.PageOK(c, list, total, page, pageSize)
}

// RescheduleAudit 改期审批
func (h *Controller) RescheduleAudit(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req dto.StaffRescheduleAuditReq
	if err := bind.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.StaffRescheduleAudit(c.Request.Context(), op, id, req.Approved, req.Remark); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

// RefundAudit 退款审核
func (h *Controller) RefundAudit(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req struct {
		Approve bool   `json:"approve"` // 是否通过（false=驳回，不使用 required 以放行布尔零值）
		Remark  string `json:"remark"`  // 审核备注
	}
	if err := bind.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.AuditRefund(c.Request.Context(), op, id, req.Approve, req.Remark); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

// ---------------------------------------------------------------------
// 日程 / 线索 / 简报
// ---------------------------------------------------------------------

// ScheduleList 日程列表（body: start_date/end_date/photographer_id）
func (h *Controller) ScheduleList(c *gin.Context) {
	op := middleware.GetOperator(c)
	photographerID := params.Int64(c, "photographer_id")
	list, err := h.Svc.ListCalendar(c.Request.Context(), op, params.Str(c, "start_date"), params.Str(c, "end_date"), photographerID)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, list)
}

// BriefGenerate 生成线索 AI 简报
func (h *Controller) BriefGenerate(c *gin.Context) {
	op := middleware.GetOperator(c)
	leadID, err := bind.PathID(c, "lead_id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	items, err := h.Svc.StaffBriefGenerate(c.Request.Context(), op, leadID)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, items)
}

// BriefList 简报项列表
func (h *Controller) BriefList(c *gin.Context) {
	op := middleware.GetOperator(c)
	leadID, err := bind.PathID(c, "lead_id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	items, err := h.Svc.StaffBriefList(c.Request.Context(), op, leadID)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, items)
}

// BriefSend 发送追问
func (h *Controller) BriefSend(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.StaffBriefSend(c.Request.Context(), op, id); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

// BriefConfirm 确认简报项
func (h *Controller) BriefConfirm(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req dto.StaffBriefConfirmReq
	if err := bind.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.StaffBriefConfirm(c.Request.Context(), op, id, req.Value); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

// ---------------------------------------------------------------------
// 定制需求 / 档期模板 / 评价 / 设置 / 设备
// ---------------------------------------------------------------------

// CustomRequestList 定制需求列表
func (h *Controller) StaffCustomRequestList(c *gin.Context) {
	op := middleware.GetOperator(c)
	page, pageSize := bind.Pager(c)
	status := params.Int(c, "status")
	list, total, err := h.Svc.StaffCustomRequests(c.Request.Context(), op, page, pageSize, status)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.PageOK(c, list, total, page, pageSize)
}

// CustomRequestRespond 响应定制需求
func (h *Controller) CustomRequestRespond(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req dto.StaffCustomRequestRespondReq
	if err := bind.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.StaffCustomRequestRespond(c.Request.Context(), op, id, req.Response); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

// ReviewList 评价列表
func (h *Controller) ReviewList(c *gin.Context) {
	op := middleware.GetOperator(c)
	page, pageSize := bind.Pager(c)
	minRating := params.Int(c, "min_rating")
	list, total, err := h.Svc.StaffReviewList(c.Request.Context(), op, page, pageSize, minRating)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.PageOK(c, list, total, page, pageSize)
}

// ReviewReply 回复评价
func (h *Controller) ReviewReply(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req dto.StaffReviewReplyReq
	if err := bind.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.StaffReviewReply(c.Request.Context(), op, id, req.Reply); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

// DeviceList 登录设备列表
func (h *Controller) DeviceList(c *gin.Context) {
	op := middleware.GetOperator(c)
	list, err := h.Svc.ListDevices(c.Request.Context(), op)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, list)
}

// DeviceRemove 踢出登录设备
func (h *Controller) DeviceRemove(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.RemoveDevice(c.Request.Context(), op, id); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

// ---------------------------------------------------------------------
// 客户档案 / 手机号换绑（报告 H7、H8）
// ---------------------------------------------------------------------

// TodayFollow 今日待跟进（到期/逾期且未成交未流失的线索）
func (h *Controller) TodayFollow(c *gin.Context) {
	op := middleware.GetOperator(c)
	list, err := h.Svc.StaffTodayFollowUp(c.Request.Context(), op, params.Int(c, "limit"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, list)
}

// CustomerMobileUpdate 修改客户手机号（换绑，含格式与占用校验）
func (h *Controller) CustomerMobileUpdate(c *gin.Context) {
	op := middleware.GetOperator(c)
	var req dto.StaffCustomerMobileReq
	if err := bind.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.UpdateCustomerMobile(c.Request.Context(), op, req.CustomerID, req.Mobile); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

// ---------------------------------------------------------------------
// 收款核验 / 订单加项（报告 H9、H10）
// ---------------------------------------------------------------------

// ---------------------------------------------------------------------
// 反馈整理（报告 H11）
// ---------------------------------------------------------------------

// FeedbackList 客户修图反馈列表（body: status 1-待处理 2-已处理，0-全部）
func (h *Controller) FeedbackList(c *gin.Context) {
	op := middleware.GetOperator(c)
	page, pageSize := bind.Pager(c)
	list, total, err := h.Svc.ListFeedbackItems(c.Request.Context(), op, params.Int(c, "status"), page, pageSize)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.PageOK(c, list, total, page, pageSize)
}

// FeedbackHandle 标记反馈已处理并记录处理备注
func (h *Controller) FeedbackHandle(c *gin.Context) {
	op := middleware.GetOperator(c)
	itemID, err := bind.PathID(c, "item_id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req dto.StaffFeedbackHandleReq
	if err := bind.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.HandleFeedbackItem(c.Request.Context(), op, itemID, req.Remark); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}
