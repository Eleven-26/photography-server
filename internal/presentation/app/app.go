package app

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"photography-server/internal/config"
	"photography-server/internal/enum"
	"photography-server/internal/middleware"
	"photography-server/internal/pkg/errs"
	"photography-server/internal/presentation/dto"
	"photography-server/internal/response"
	"photography-server/internal/service"
)

// Controller 摄影师 App 端接口（订单处理 + 日程 + 线索 AI 简报 + 个人中心）。
// 员工身份经 StaffAuth 注入 Operator；订单状态流转/收款/交付等复杂操作
// 复用 PC 端既有 service 方法（同一业务规则，两端入口分开）。
type Controller struct {
	Svc *service.Service
	Cfg *config.Config
}

func New(svc *service.Service, cfg *config.Config) *Controller {
	return &Controller{Svc: svc, Cfg: cfg}
}

// RegisterPublic 注册公开路由（验证码登录）
func (h *Controller) RegisterPublic(g *gin.RouterGroup) {
	g.POST("/auth/sms-code", h.SmsCode)
	g.POST("/auth/login", h.Login)
}

// RegisterAuthed 注册需登录路由（StaffAuth 注入 Operator）
func (h *Controller) RegisterAuthed(g *gin.RouterGroup) {
	// 工作台
	g.POST("/overview", h.Overview)
	// 订单（复用 PC 端 service）
	g.POST("/order/list", h.OrderList)
	g.POST("/order/detail/:id", h.OrderDetail)
	g.POST("/order/status/:id", h.OrderStatus)
	g.POST("/order/logs/:id", h.OrderLogs)
	g.POST("/order/create", h.OrderCreate)
	// 收款（拍照上传凭证 → 工作室核验）
	g.POST("/payment/create/:order_id", h.PaymentCreate)
	g.POST("/payment/list/:order_id", h.PaymentList)
	// 交付
	g.POST("/delivery/detail/:id", h.DeliveryDetail)
	g.POST("/delivery/create/:order_id", h.DeliveryCreate)
	g.POST("/delivery/upload-samples/:id", h.DeliveryUploadSamples)
	g.POST("/delivery/upload-retouched/:id", h.DeliveryUploadRetouched)
	// 改期审批
	g.POST("/reschedule/list", h.RescheduleList)
	g.POST("/reschedule/audit/:id", h.RescheduleAudit)
	// 退款（查看/审核复用 PC 端）
	g.POST("/refund/list/:order_id", h.RefundList)
	g.POST("/refund/audit/:id", h.RefundAudit)
	// 日程
	g.POST("/schedule/list", h.ScheduleList)
	// 线索跟进 + AI 简报
	g.POST("/lead/list", h.LeadList)
	g.POST("/lead/detail/:id", h.LeadDetail)
	g.POST("/lead/messages/:id", h.LeadMessages)
	g.POST("/lead/message/send/:id", h.LeadMessageSend)
	g.POST("/brief/generate/:lead_id", h.BriefGenerate)
	g.POST("/brief/list/:lead_id", h.BriefList)
	g.POST("/brief/send/:id", h.BriefSend)
	g.POST("/brief/confirm/:id", h.BriefConfirm)
	// 定制需求
	g.POST("/custom-request/list", h.CustomRequestList)
	g.POST("/custom-request/respond/:id", h.CustomRequestRespond)
	// 档期时段模板
	g.POST("/slot-template/list", h.SlotTemplateList)
	g.POST("/slot-template/save", h.SlotTemplateSave)
	g.POST("/slot-template/save/:id", h.SlotTemplateSave)
	g.POST("/slot-template/delete/:id", h.SlotTemplateDelete)
	// 评价
	g.POST("/review/list", h.ReviewList)
	g.POST("/review/reply/:id", h.ReviewReply)
	// 工作室设置
	g.POST("/studio/get", h.StudioGet)
	g.POST("/studio/update", h.StudioUpdate)
	// 个人中心（设备管理）
	g.POST("/device/list", h.DeviceList)
	g.POST("/device/remove/:id", h.DeviceRemove)
}

func (h *Controller) bindJSON(c *gin.Context, obj interface{}) error {
	if err := c.ShouldBindJSON(obj); err != nil {
		return errs.BadRequest(errs.ErrBadRequest + "：" + err.Error())
	}
	return nil
}

func pager(c *gin.Context) (int, int) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 10
	}
	return page, pageSize
}

func pathID(c *gin.Context, name string) (int64, error) {
	id, err := strconv.ParseInt(c.Param(name), 10, 64)
	if err != nil || id <= 0 {
		return 0, errs.BadRequest("参数错误")
	}
	return id, nil
}

// ---------------------------------------------------------------------
// 公开接口
// ---------------------------------------------------------------------

// SmsCode 发送登录验证码
func (h *Controller) SmsCode(c *gin.Context) {
	var req struct {
		Mobile string `json:"mobile" binding:"required"`
	}
	if err := h.bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.SendSmsCode(c.Request.Context(), "login", req.Mobile); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

// Login 摄影师手机号验证码登录（按 sys_user.mobile 匹配员工）
func (h *Controller) Login(c *gin.Context) {
	var req struct {
		Mobile     string `json:"mobile" binding:"required"`
		Code       string `json:"code" binding:"required"`
		DeviceName string `json:"device_name"` // 设备名称（选填，用于设备管理）
		Platform   string `json:"platform"`    // ios/android
	}
	if err := h.bindJSON(c, &req); err != nil {
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
	ov, err := h.Svc.AppOverview(c.Request.Context(), op)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, ov)
}

// OrderList 订单列表（query: status）
func (h *Controller) OrderList(c *gin.Context) {
	op := middleware.GetOperator(c)
	page, pageSize := pager(c)
	list, total, err := h.Svc.ListOrders(c.Request.Context(), op, page, pageSize, c.Query("status"), 0)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{"list": list, "total": total})
}

// OrderDetail 订单详情
func (h *Controller) OrderDetail(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := pathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	detail, err := h.Svc.GetOrderDetail(c.Request.Context(), op, id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, detail)
}

// OrderCreate 创建订单
func (h *Controller) OrderCreate(c *gin.Context) {
	op := middleware.GetOperator(c)
	var req dto.OrderCreateReq
	if err := h.bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	o, err := h.Svc.CreateOrder(c.Request.Context(), op, req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, o)
}

// OrderStatus 订单状态流转
func (h *Controller) OrderStatus(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := pathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req dto.OrderStatusReq
	if err := h.bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.ChangeOrderStatus(c.Request.Context(), op, id, enum.OrderStatus(req.Status), req.Content); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

// OrderLogs 订单操作日志
func (h *Controller) OrderLogs(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := pathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	detail, err := h.Svc.GetOrderDetail(c.Request.Context(), op, id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, detail.Logs)
}

// ---------------------------------------------------------------------
// 收款 / 交付 / 改期 / 退款
// ---------------------------------------------------------------------

// PaymentCreate 录入收款（App 拍凭证上传）
func (h *Controller) PaymentCreate(c *gin.Context) {
	op := middleware.GetOperator(c)
	orderID, err := pathID(c, "order_id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req dto.PaymentCreateReq
	if err := h.bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	p, err := h.Svc.CreatePayment(c.Request.Context(), op, orderID, req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, p)
}

// PaymentList 收款记录
func (h *Controller) PaymentList(c *gin.Context) {
	op := middleware.GetOperator(c)
	orderID, err := pathID(c, "order_id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	list, err := h.Svc.ListPayments(c.Request.Context(), op, orderID)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, list)
}

// DeliveryDetail 交付单明细
func (h *Controller) DeliveryDetail(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := pathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	d, err := h.Svc.GetDeliveryByOrder(c.Request.Context(), op, id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, d)
}

// DeliveryCreate 创建交付单
func (h *Controller) DeliveryCreate(c *gin.Context) {
	op := middleware.GetOperator(c)
	orderID, err := pathID(c, "order_id")
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

// DeliveryUploadSamples 上传样片
func (h *Controller) DeliveryUploadSamples(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := pathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req struct {
		Items []dto.DeliveryItemReq `json:"items" binding:"required"`
	}
	if err := h.bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.UploadSamples(c.Request.Context(), op, id, req.Items); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

// DeliveryUploadRetouched 上传精修成品
func (h *Controller) DeliveryUploadRetouched(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := pathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req struct {
		Items []dto.DeliveryItemReq `json:"items" binding:"required"`
	}
	if err := h.bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.UploadRetouched(c.Request.Context(), op, id, req.Items); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

// RescheduleList 改期单列表
func (h *Controller) RescheduleList(c *gin.Context) {
	op := middleware.GetOperator(c)
	page, pageSize := pager(c)
	status, _ := strconv.Atoi(c.Query("status"))
	list, total, err := h.Svc.AppRescheduleList(c.Request.Context(), op, page, pageSize, status)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{"list": list, "total": total})
}

// RescheduleAudit 改期审批
func (h *Controller) RescheduleAudit(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := pathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req dto.AppRescheduleAuditReq
	if err := h.bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.AppRescheduleAudit(c.Request.Context(), op, id, req.Approved, req.Remark); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

// RefundList 退款记录
func (h *Controller) RefundList(c *gin.Context) {
	op := middleware.GetOperator(c)
	orderID, err := pathID(c, "order_id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	list, err := h.Svc.ListRefunds(c.Request.Context(), op, orderID)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, list)
}

// RefundAudit 退款审核
func (h *Controller) RefundAudit(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := pathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req struct {
		Approve bool   `json:"approve" binding:"required"` // 是否通过
		Remark  string `json:"remark"`                     // 审核备注
	}
	if err := h.bindJSON(c, &req); err != nil {
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

// ScheduleList 日程列表（query: start_date/end_date/photographer_id）
func (h *Controller) ScheduleList(c *gin.Context) {
	op := middleware.GetOperator(c)
	photographerID, _ := strconv.ParseInt(c.Query("photographer_id"), 10, 64)
	list, err := h.Svc.ListCalendar(c.Request.Context(), op, c.Query("start_date"), c.Query("end_date"), photographerID)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, list)
}

// LeadList 线索列表
func (h *Controller) LeadList(c *gin.Context) {
	op := middleware.GetOperator(c)
	page, pageSize := pager(c)
	ownerID, _ := strconv.ParseInt(c.Query("owner_id"), 10, 64)
	list, total, err := h.Svc.ListLeads(c.Request.Context(), op, page, pageSize, c.Query("keyword"), c.Query("status"), ownerID)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{"list": list, "total": total})
}

// LeadDetail 线索详情
func (h *Controller) LeadDetail(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := pathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	l, err := h.Svc.GetLeadDetail(c.Request.Context(), op, id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, l)
}

// LeadMessages 线索沟通记录
func (h *Controller) LeadMessages(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := pathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	list, err := h.Svc.AppLeadMessages(c.Request.Context(), op, id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, list)
}

// LeadMessageSend 发送线索沟通消息
func (h *Controller) LeadMessageSend(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := pathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req dto.AppLeadMessageReq
	if err := h.bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	m, err := h.Svc.AppSendLeadMessage(c.Request.Context(), op, id, req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, m)
}

// BriefGenerate 生成线索 AI 简报
func (h *Controller) BriefGenerate(c *gin.Context) {
	op := middleware.GetOperator(c)
	leadID, err := pathID(c, "lead_id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	items, err := h.Svc.AppBriefGenerate(c.Request.Context(), op, leadID)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, items)
}

// BriefList 简报项列表
func (h *Controller) BriefList(c *gin.Context) {
	op := middleware.GetOperator(c)
	leadID, err := pathID(c, "lead_id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	items, err := h.Svc.AppBriefList(c.Request.Context(), op, leadID)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, items)
}

// BriefSend 发送追问
func (h *Controller) BriefSend(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := pathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.AppBriefSend(c.Request.Context(), op, id); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

// BriefConfirm 确认简报项
func (h *Controller) BriefConfirm(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := pathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req dto.AppBriefConfirmReq
	if err := h.bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.AppBriefConfirm(c.Request.Context(), op, id, req.Value); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

// ---------------------------------------------------------------------
// 定制需求 / 档期模板 / 评价 / 设置 / 设备
// ---------------------------------------------------------------------

// CustomRequestList 定制需求列表
func (h *Controller) CustomRequestList(c *gin.Context) {
	op := middleware.GetOperator(c)
	page, pageSize := pager(c)
	status, _ := strconv.Atoi(c.Query("status"))
	list, total, err := h.Svc.AppCustomRequests(c.Request.Context(), op, page, pageSize, status)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{"list": list, "total": total})
}

// CustomRequestRespond 响应定制需求
func (h *Controller) CustomRequestRespond(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := pathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req dto.AppCustomRequestRespondReq
	if err := h.bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.AppCustomRequestRespond(c.Request.Context(), op, id, req.Response); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

// SlotTemplateList 档期时段模板列表
func (h *Controller) SlotTemplateList(c *gin.Context) {
	op := middleware.GetOperator(c)
	photographerID, _ := strconv.ParseInt(c.Query("photographer_id"), 10, 64)
	list, err := h.Svc.AppSlotTemplates(c.Request.Context(), op, photographerID)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, list)
}

// SlotTemplateSave 新建/更新档期时段模板
func (h *Controller) SlotTemplateSave(c *gin.Context) {
	op := middleware.GetOperator(c)
	var id int64
	if raw := c.Param("id"); raw != "" {
		parsed, err := pathID(c, "id")
		if err != nil {
			response.Fail(c, err)
			return
		}
		id = parsed
	}
	var req dto.AppSlotTemplateReq
	if err := h.bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	m, err := h.Svc.AppSaveSlotTemplate(c.Request.Context(), op, id, req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, m)
}

// SlotTemplateDelete 删除档期时段模板
func (h *Controller) SlotTemplateDelete(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := pathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.AppDeleteSlotTemplate(c.Request.Context(), op, id); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

// ReviewList 评价列表
func (h *Controller) ReviewList(c *gin.Context) {
	op := middleware.GetOperator(c)
	page, pageSize := pager(c)
	minRating, _ := strconv.Atoi(c.Query("min_rating"))
	list, total, err := h.Svc.AppReviewList(c.Request.Context(), op, page, pageSize, minRating)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{"list": list, "total": total})
}

// ReviewReply 回复评价
func (h *Controller) ReviewReply(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := pathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req dto.AppReviewReplyReq
	if err := h.bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.AppReviewReply(c.Request.Context(), op, id, req.Reply); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

// StudioGet 工作室设置
func (h *Controller) StudioGet(c *gin.Context) {
	op := middleware.GetOperator(c)
	st, err := h.Svc.AppStudioSettingGet(c.Request.Context(), op)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, st)
}

// StudioUpdate 工作室设置更新
func (h *Controller) StudioUpdate(c *gin.Context) {
	op := middleware.GetOperator(c)
	var req dto.AppStudioSettingReq
	if err := h.bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	updates := req.ToUpdates()
	if err := h.Svc.AppStudioSettingUpdate(c.Request.Context(), op, updates); err != nil {
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
	id, err := pathID(c, "id")
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
