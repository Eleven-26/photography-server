package h5

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"photography-server/internal/config"
	"photography-server/internal/middleware"
	"photography-server/internal/pkg/errs"
	"photography-server/internal/pkg/params"
	"photography-server/internal/presentation/dto"
	"photography-server/internal/response"
	"photography-server/internal/service"
)

// Controller 客户 H5 端接口（客户预约全链路：浏览套餐 → 提交预约 → 支付定金 →
// 选片 → 确认成片 → 评价）。公开接口无需登录，业务接口经 CustomerAuth 注入客户上下文。
type Controller struct {
	Svc *service.Service
	Cfg *config.Config
}

func New(svc *service.Service, cfg *config.Config) *Controller {
	return &Controller{Svc: svc, Cfg: cfg}
}

// RegisterPublic 注册公开路由（无需登录；租户由 slug 短链标识定位——query slug / X-Slug 头，
// 服务端反查 company_id，#29 移除裸 company_id 防遍历枚举）
func (h *Controller) RegisterPublic(g *gin.RouterGroup) {
	g.POST("/auth/sms-code", h.SmsCode)
	g.POST("/auth/login", h.Login)
	g.POST("/package/list", h.PackageList)
	g.POST("/package/detail/:id", h.PackageDetail)
	g.POST("/studio/info", h.StudioInfo)
	g.POST("/slot/list", h.SlotList)
	g.POST("/custom-request/submit", h.CustomRequestSubmit)
	// 作品集（报告 H5）：预约主页展示，只出「已发布 + 公开」作品
	g.POST("/asset/list", h.AssetList)
	g.POST("/asset/detail/:id", h.AssetDetail)
}

// RegisterAuthed 注册需登录路由（CustomerAuth 注入 ClientUser）
func (h *Controller) RegisterAuthed(g *gin.RouterGroup) {
	// 预约/订单
	g.POST("/order/submit", h.BookingSubmit)
	g.POST("/order/confirm/:id", h.BookingConfirm)
	g.POST("/order/cancel/:id", h.BookingCancel)
	g.POST("/order/list", h.OrderList)
	g.POST("/order/detail/:id", h.OrderDetail)
	// 改期
	g.POST("/reschedule/apply/:order_id", h.RescheduleApply)
	g.POST("/reschedule/cancel/:id", h.RescheduleCancel)
	// 退款
	g.POST("/refund/apply/:order_id", h.RefundApply)
	// 评价
	g.POST("/review/create/:order_id", h.ReviewCreate)
	// 选片与交付
	g.POST("/delivery/detail/:id", h.DeliveryDetail)
	g.POST("/delivery/select/:id", h.SelectPhotos)
	g.POST("/delivery/confirm-extra/:id", h.ConfirmExtra)
	g.POST("/delivery/confirm/:id", h.ConfirmDelivery)
	g.POST("/delivery/feedback/:item_id", h.FeedbackSubmit)
	// 定制需求
	g.POST("/custom-request/list", h.CustomRequestList)
	// 报价（报告 H1）：查看 / 接受 / 提出修改
	g.POST("/quote/list", h.QuoteList)
	g.POST("/quote/accept/:id", h.QuoteAccept)
	g.POST("/quote/modify/:id", h.QuoteModify)
	// 改期调度费（报告 H2）：详情含支付状态，支付走「上传凭证 → 工作室核验」
	g.POST("/reschedule/detail/:id", h.RescheduleDetail)
	g.POST("/reschedule/pay/:id", h.ReschedulePay)
	// 加片费试算（报告 H3）
	g.POST("/delivery/extra-quote/:id", h.ExtraQuote)
	// 拍摄需求修改（报告 H4）
	g.POST("/order/requirement/update/:id", h.OrderRequirementUpdate)
	// 站内通知（报告 H6）
	g.POST("/notification/list", h.NotificationList)
	g.POST("/notification/unread-count", h.NotificationUnreadCount)
	g.POST("/notification/read/:id", h.NotificationRead)
	g.POST("/notification/read-all", h.NotificationReadAll)
}

// slugFrom 提取客户端公开接口的预约主页短链标识：query slug 与 X-Slug 头二选一（头优先）。
// slug 形如 "sunset-studio"，为工作室预约主页 URL/二维码携带的不可枚举标识（#29），
// 服务端据此反查 company_id，绝不接受客户端直传裸 company_id。
func slugFrom(c *gin.Context) string {
	if v := c.GetHeader("X-Slug"); v != "" {
		return v
	}
	// POST 参数统一走 body；query 仅作预约主页短链的兜底
	if v := params.Str(c, "slug"); v != "" {
		return v
	}
	return c.Query("slug")
}

// requireCompany 按 slug 反查并校验租户（数据库不存在/未配置 slug 时返回业务错误）
func (h *Controller) requireCompany(c *gin.Context) (int64, error) {
	slug := slugFrom(c)
	if slug == "" {
		return 0, errs.BadRequest("缺少工作室标识（slug）")
	}
	companyID, err := h.Svc.ResolveCompanyBySlug(c.Request.Context(), slug)
	if err != nil {
		return 0, errs.Internal("")
	}
	if companyID <= 0 {
		return 0, errs.BadRequest("预约主页不存在或未配置短链标识，请联系工作室")
	}
	return companyID, nil
}

// bindJSON 绑定 JSON 请求体
func (h *Controller) bindJSON(c *gin.Context, obj interface{}) error {
	if err := c.ShouldBindJSON(obj); err != nil {
		return errs.BadRequest(errs.ErrBadRequest + "：" + err.Error())
	}
	return nil
}

func pager(c *gin.Context) (int, int) {
	page := params.Int(c, "page")
	pageSize := params.Int(c, "page_size")
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

// Login 客户手机号验证码登录（未注册自动建档），openid 为小程序场景透传。
// 租户定位：body slug（可选）→ query slug / X-Slug 头 → 服务端反查 company_id（#29）
func (h *Controller) Login(c *gin.Context) {
	var req struct {
		Slug   string `json:"slug"` // 预约主页短链标识（也可放 query/头）
		Mobile string `json:"mobile" binding:"required"`
		Code   string `json:"code" binding:"required"`
		OpenID string `json:"openid"`
	}
	if err := h.bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	slug := req.Slug
	if slug == "" {
		slug = slugFrom(c)
	}
	if slug == "" {
		response.Fail(c, errs.BadRequest("缺少工作室标识（slug）"))
		return
	}
	companyID, err := h.Svc.ResolveCompanyBySlug(c.Request.Context(), slug)
	if err != nil {
		response.Fail(c, errs.Internal(""))
		return
	}
	if companyID <= 0 {
		response.Fail(c, errs.BadRequest("预约主页不存在或未配置短链标识，请联系工作室"))
		return
	}
	customer, token, err := h.Svc.CustomerSmsLogin(c.Request.Context(), companyID, req.Mobile, req.Code, req.OpenID)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{
		"token":    token,
		"customer": customer,
	})
}

// PackageList 套餐列表（已上架）
func (h *Controller) PackageList(c *gin.Context) {
	companyID, err := h.requireCompany(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	page, pageSize := pager(c)
	list, total, err := h.Svc.ClientPackages(c.Request.Context(), companyID, page, pageSize, params.Str(c, "category"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{"list": list, "total": total})
}

// PackageDetail 套餐详情
func (h *Controller) PackageDetail(c *gin.Context) {
	companyID, err := h.requireCompany(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	id, err := pathID(c, "id")
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

// CustomRequestSubmit 提交定制需求（游客/登录均可）
func (h *Controller) CustomRequestSubmit(c *gin.Context) {
	var req dto.ClientCustomRequestReq
	if err := h.bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	cu := middleware.GetClientUser(c)
	var companyID int64
	if cu != nil {
		// 已登录：租户取令牌内绑定的公司（可信，不信任客户端请求参数）
		companyID = cu.CompanyID
	} else {
		// 游客：按 slug 反查（#29）
		var err error
		companyID, err = h.requireCompany(c)
		if err != nil {
			response.Fail(c, err)
			return
		}
	}
	m, err := h.Svc.ClientSubmitCustomRequest(c.Request.Context(), companyID, cu, req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, m)
}

// ---------------------------------------------------------------------
// 登录后接口
// ---------------------------------------------------------------------

// BookingSubmit 提交预约单
func (h *Controller) BookingSubmit(c *gin.Context) {
	cu := middleware.GetClientUser(c)
	var req dto.ClientBookingReq
	if err := h.bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	o, err := h.Svc.ClientSubmitBooking(c.Request.Context(), cu, req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, o)
}

// BookingConfirm 确认预约单
func (h *Controller) BookingConfirm(c *gin.Context) {
	cu := middleware.GetClientUser(c)
	id, err := pathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.ClientConfirmBooking(c.Request.Context(), cu, id); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

// BookingCancel 取消预约单
func (h *Controller) BookingCancel(c *gin.Context) {
	cu := middleware.GetClientUser(c)
	id, err := pathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req struct {
		Reason string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&req)
	if err := h.Svc.ClientCancelBooking(c.Request.Context(), cu, id, req.Reason); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

// OrderList 我的订单
func (h *Controller) OrderList(c *gin.Context) {
	cu := middleware.GetClientUser(c)
	page, pageSize := pager(c)
	list, total, err := h.Svc.ClientOrders(c.Request.Context(), cu, page, pageSize, params.Str(c, "status"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{"list": list, "total": total})
}

// OrderDetail 订单详情
func (h *Controller) OrderDetail(c *gin.Context) {
	cu := middleware.GetClientUser(c)
	id, err := pathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	detail, err := h.Svc.ClientOrderDetail(c.Request.Context(), cu, id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, detail)
}

// RescheduleApply 申请改期
func (h *Controller) RescheduleApply(c *gin.Context) {
	cu := middleware.GetClientUser(c)
	orderID, err := pathID(c, "order_id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req dto.ClientRescheduleReq
	if err := h.bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	rs, err := h.Svc.ClientRescheduleApply(c.Request.Context(), cu, orderID, req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, rs)
}

// RescheduleCancel 撤回改期申请
func (h *Controller) RescheduleCancel(c *gin.Context) {
	cu := middleware.GetClientUser(c)
	id, err := pathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.ClientRescheduleCancel(c.Request.Context(), cu, id); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

// RefundApply 申请退款
func (h *Controller) RefundApply(c *gin.Context) {
	cu := middleware.GetClientUser(c)
	orderID, err := pathID(c, "order_id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req dto.ClientRefundReq
	if err := h.bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	rf, err := h.Svc.ClientRefundApply(c.Request.Context(), cu, orderID, req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, rf)
}

// ReviewCreate 评价订单
func (h *Controller) ReviewCreate(c *gin.Context) {
	cu := middleware.GetClientUser(c)
	orderID, err := pathID(c, "order_id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req dto.ClientReviewReq
	if err := h.bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	rv, err := h.Svc.ClientReviewCreate(c.Request.Context(), cu, orderID, req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, rv)
}

// DeliveryDetail 交付单与明细（选片页/成片页）
func (h *Controller) DeliveryDetail(c *gin.Context) {
	cu := middleware.GetClientUser(c)
	id, err := pathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	d, items, err := h.Svc.ClientDeliveryItems(c.Request.Context(), cu, id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{"delivery": d, "items": items})
}

// SelectPhotos 提交选片
func (h *Controller) SelectPhotos(c *gin.Context) {
	cu := middleware.GetClientUser(c)
	id, err := pathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req struct {
		ItemIDs []int64 `json:"item_ids" binding:"required"`
	}
	if err := h.bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.ClientSelectPhotos(c.Request.Context(), cu, id, req.ItemIDs); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

// ConfirmExtra 确认加片费用
func (h *Controller) ConfirmExtra(c *gin.Context) {
	cu := middleware.GetClientUser(c)
	id, err := pathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.ClientConfirmExtra(c.Request.Context(), cu, id); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

// ConfirmDelivery 确认成片
func (h *Controller) ConfirmDelivery(c *gin.Context) {
	cu := middleware.GetClientUser(c)
	id, err := pathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.ClientConfirmDelivery(c.Request.Context(), cu, id); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

// FeedbackSubmit 提交精修反馈
func (h *Controller) FeedbackSubmit(c *gin.Context) {
	cu := middleware.GetClientUser(c)
	itemID, err := pathID(c, "item_id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req dto.ClientFeedbackReq
	if err := h.bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.ClientFeedbackSubmit(c.Request.Context(), cu, itemID, req); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

// CustomRequestList 我的定制需求
func (h *Controller) CustomRequestList(c *gin.Context) {
	cu := middleware.GetClientUser(c)
	page, pageSize := pager(c)
	list, total, err := h.Svc.ClientCustomRequests(c.Request.Context(), cu, page, pageSize)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{"list": list, "total": total})
}

// ---------------------------------------------------------------------
// 作品集（报告 H5，公开接口）
// ---------------------------------------------------------------------

// AssetList 公开作品列表（预约主页作品集）
func (h *Controller) AssetList(c *gin.Context) {
	companyID, err := h.requireCompany(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	page, pageSize := pager(c)
	list, total, err := h.Svc.ClientAssets(c.Request.Context(), companyID, page, pageSize,
		params.Str(c, "category"), params.Str(c, "featured") == "1")
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{"list": list, "total": total})
}

// AssetDetail 公开作品详情（浏览数 +1）
func (h *Controller) AssetDetail(c *gin.Context) {
	companyID, err := h.requireCompany(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	id, err := pathID(c, "id")
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

// ---------------------------------------------------------------------
// 报价（报告 H1）
// ---------------------------------------------------------------------

// QuoteList 我的报价单列表（含明细字段，前端按 id 取单条即可）
func (h *Controller) QuoteList(c *gin.Context) {
	cu := middleware.GetClientUser(c)
	list, err := h.Svc.ClientQuotes(c.Request.Context(), cu)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, list)
}

// QuoteAccept 接受报价
func (h *Controller) QuoteAccept(c *gin.Context) {
	cu := middleware.GetClientUser(c)
	id, err := pathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.ClientQuoteAccept(c.Request.Context(), cu, id); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

// QuoteModify 对报价提出修改意见
func (h *Controller) QuoteModify(c *gin.Context) {
	cu := middleware.GetClientUser(c)
	id, err := pathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req dto.ClientQuoteModifyReq
	if err := h.bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.ClientQuoteModify(c.Request.Context(), cu, id, req.Content); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

// ---------------------------------------------------------------------
// 调度费 / 加片费试算 / 需求修改（报告 H2~H4）
// ---------------------------------------------------------------------

// RescheduleDetail 改期单详情 + 调度费支付状态
func (h *Controller) RescheduleDetail(c *gin.Context) {
	cu := middleware.GetClientUser(c)
	id, err := pathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	d, err := h.Svc.ClientRescheduleDetail(c.Request.Context(), cu, id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, d)
}

// ReschedulePay 提交改期调度费支付凭证
func (h *Controller) ReschedulePay(c *gin.Context) {
	cu := middleware.GetClientUser(c)
	id, err := pathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req dto.ClientReschedulePayReq
	if err := h.bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	p, err := h.Svc.ClientPayRescheduleFee(c.Request.Context(), cu, id, req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, p)
}

// ExtraQuote 加片费试算（body 可为空：按当前已选张数试算）
func (h *Controller) ExtraQuote(c *gin.Context) {
	cu := middleware.GetClientUser(c)
	id, err := pathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req dto.ClientExtraQuoteReq
	_ = c.ShouldBindJSON(&req)
	q, err := h.Svc.ClientExtraQuote(c.Request.Context(), cu, id, req.SelectCount)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, q)
}

// OrderRequirementUpdate 客户修改拍摄需求（仅待定金/待拍摄，白名单字段）
func (h *Controller) OrderRequirementUpdate(c *gin.Context) {
	cu := middleware.GetClientUser(c)
	id, err := pathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req dto.ClientOrderRequirementReq
	if err := h.bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.ClientUpdateOrderRequirement(c.Request.Context(), cu, id, req); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

// ---------------------------------------------------------------------
// 站内通知（报告 H6）
// ---------------------------------------------------------------------

// NotificationList 我的通知列表（body: unread=1 只看未读）
func (h *Controller) NotificationList(c *gin.Context) {
	cu := middleware.GetClientUser(c)
	page, pageSize := pager(c)
	list, total, err := h.Svc.ListClientNotifications(c.Request.Context(), cu, page, pageSize, params.Str(c, "unread") == "1")
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{"list": list, "total": total})
}

// NotificationUnreadCount 未读通知数（铃铛红点）
func (h *Controller) NotificationUnreadCount(c *gin.Context) {
	cu := middleware.GetClientUser(c)
	count, err := h.Svc.UnreadClientNotificationCount(c.Request.Context(), cu)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{"count": count})
}

// NotificationRead 标记单条已读
func (h *Controller) NotificationRead(c *gin.Context) {
	cu := middleware.GetClientUser(c)
	id, err := pathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.MarkClientNotificationRead(c.Request.Context(), cu, id); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

// NotificationReadAll 全部标记已读
func (h *Controller) NotificationReadAll(c *gin.Context) {
	cu := middleware.GetClientUser(c)
	if err := h.Svc.MarkAllClientNotificationsRead(c.Request.Context(), cu); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}
