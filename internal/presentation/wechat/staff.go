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

// RegisterStaffPublic 注册员工区公开路由（挂 /wechat/staff）。
//
// 员工端**登录方式集合**（2026-09-14：主登录方式由「手机号+验证码」改为「账号+密码」，与 PC 一致）：
//
//	auth/login          账号+密码登录   ← 当前唯一被前端调用的方式，复用 PC 同一套凭据校验与失败锁定
//	auth/sms-code       发送短信验证码  ← 保留（前端暂不调用）
//	auth/login-by-code  手机验证码登录  ← 预留（后端已就绪，员工端 UI 未接入）
//	（规划）auth/login-by-wechat        微信授权登录
//
// 三条路径全部公开、不挂 StaffAuth —— 登录本身发生在拿到令牌之前。
// 注意：sms-code 与 login-by-code 是一对，只留发码接口而摘掉登录接口等于能力残缺，
// 故两者一并保留；将来接验证码登录时前端直接调它们即可，无需再动后端。
func (h *Controller) RegisterStaffPublic(g *gin.RouterGroup) {
	g.POST("/auth/sms-code", h.StaffSmsCode)
	g.POST("/auth/login", h.StaffPasswordLogin)
	g.POST("/auth/login-by-code", h.StaffLoginByCode)
}

// ---------------------------------------------------------------------
// 公开接口
// ---------------------------------------------------------------------

// StaffSmsCode 发送登录验证码（保留：员工端 UI 当前未调用，待手机验证码登录上线）
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

// StaffPasswordLogin 员工账号密码登录（与 PC 同一套凭据校验：bcrypt + 失败锁定）。
//
// 响应结构与 PC 的 /auth/login 完全一致：{ token, user: UserInfoVO }，
// user 内嵌 sys_user 全字段（含 id/username/nickname/avatar/mobile/role_id/store_id）
// 并追加 role_code / role_name / data_scope / permissions。
func (h *Controller) StaffPasswordLogin(c *gin.Context) {
	var req struct {
		Username   string `json:"username" binding:"required"` // 登录账号
		Password   string `json:"password" binding:"required"` // 登录密码
		DeviceName string `json:"device_name"`                 // 设备名称（选填，用于登录设备管理）
		Platform   string `json:"platform"`                    // ios/android
	}
	if err := bind.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	resp, err := h.Svc.StaffPasswordLogin(c.Request.Context(), req.Username, req.Password, req.DeviceName, req.Platform, c.ClientIP())
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, resp)
}

// StaffLoginByCode 员工手机号验证码登录（预留：员工端 UI 未接入，契约已就位）
func (h *Controller) StaffLoginByCode(c *gin.Context) {
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
	resp, err := h.Svc.StaffSmsLogin(c.Request.Context(), req.Mobile, req.Code, req.DeviceName, req.Platform, c.ClientIP())
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, resp)
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
