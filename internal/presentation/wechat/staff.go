package wechat

import (
	"github.com/gin-gonic/gin"

	"photography-server/internal/contract"
	"photography-server/internal/middleware"
	"photography-server/internal/pkg/params"
	"photography-server/internal/pkg/response"
	"photography-server/internal/presentation/bind"
)

// 小程序员工区接口（订单处理 + 日程 + 线索 AI 简报 + 个人中心）。
// 员工（摄影师/助理）通过小程序处理业务，身份经 StaffAuth 注入 Operator。
//
// 路由注册（2026-09-12 路由整理后）已迁到 router/endpoints.go 的 staffEndpoint：
// 与 PC 同路径的路由直接复用管理端 handler（同一函数指针，表在 routes 包），
// 本文件只保留【员工端独有 handler】与【真实端差异实现】（退款审核 approve / 创建交付单不收 body）。
// Controller 类型与 New 构造见 wechat.go。

// 路由注册已迁出本文件（2026-09-16）：
// 员工端三条登录出口（auth/sms-code、auth/login、auth/login-by-code）现声明在
// router/endpoints.go 的 staffPublicExtra —— 它们必须在拿到令牌之前可用，故单独归为
// 「免端级中间件」的路由，而不是内联在本文件里 g.POST。
// 与 PC 同路径的路由见 staffInclude，端独有的见 staffExtra；本文件从此只保留 handler。

// ---------------------------------------------------------------------
// 公开接口
// ---------------------------------------------------------------------

// StaffSmsCode 发送登录验证码（保留：员工端 UI 当前未调用，待手机验证码登录上线）
// @Summary      发送登录验证码
// @Description  员工登录短信验证码，场景固定为 login。**免鉴权**（登录发生在拿到令牌之前）。
// @Description  前端暂未调用：员工端主登录方式为「账号 + 密码」（/wechat/staff/auth/login）。
// @Tags         员工端·登录
// @Accept       json
// @Produce      json
// @Param        req  body  object{mobile=string}  true  "手机号"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Router       /wechat/staff/auth/sms-code [post]
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
// @Summary      员工账号密码登录
// @Description  员工端主登录方式，与 PC 端 /auth/login 共用同一套凭据校验（bcrypt + 失败锁定）。**免鉴权**。
// @Description  响应结构与 PC 完全一致：{ token, user }，user 内嵌 sys_user 字段并追加 role_code / role_name / data_scope / permissions。
// @Tags         员工端·登录
// @Accept       json
// @Produce      json
// @Param        req  body  object{username=string,password=string,device_name=string,platform=string}  true  "登录凭据（platform: ios/android，选填）"
// @Success      200  {object}  response.Body{data=contract.LoginResp}
// @Failure      400  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Router       /wechat/staff/auth/login [post]
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
// @Summary      员工验证码登录
// @Description  手机号 + 短信验证码登录。**免鉴权**；后端已就绪，员工端 UI 暂未接入（与 auth/sms-code 成对保留）。
// @Tags         员工端·登录
// @Accept       json
// @Produce      json
// @Param        req  body  object{mobile=string,code=string,device_name=string,platform=string}  true  "手机号与验证码（platform: ios/android，选填）"
// @Success      200  {object}  response.Body{data=contract.LoginResp}
// @Failure      400  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Router       /wechat/staff/auth/login-by-code [post]
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
// @Summary      工作台待办
// @Description  员工端首页待办统计（今日拍摄 / 待处理事项等）。
// @Tags         员工端·工作台
// @Accept       json
// @Produce      json
// @Success      200  {object}  response.Body{data=contract.StaffOverview}
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /wechat/staff/overview [post]
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
// @Summary      创建交付单
// @Description  **端差异实现**：员工端只接 body.order_id（不传 stage），与 PC 的 /delivery/create 契约不同，两个前端各自依赖，不可合并。
// @Tags         员工端·交付
// @Accept       json
// @Produce      json
// @Param        req  body  object{order_id=int}  true  "订单ID"
// @Success      200  {object}  response.Body{data=model.Delivery}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /wechat/staff/delivery/create [post]
func (h *Controller) DeliveryCreate(c *gin.Context) {
	op := middleware.GetOperator(c)
	orderID, err := bind.BodyID(c, "order_id")
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
// @Summary      改期单列表
// @Description  分页查询改期单，可按状态过滤。（发起改期复用 PC 的 /order/reschedule/apply/:order_id）
// @Tags         员工端·改期
// @Accept       json
// @Produce      json
// @Param        req  body  object{page=int,page_size=int,status=int}  true  "查询条件（status 0=全部）"
// @Success      200  {object}  response.Body{data=response.Page{list=[]model.OrderReschedule,total=int,page=int,page_size=int}}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /wechat/staff/reschedule/list [post]
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
// @Summary      改期审批
// @Tags         员工端·改期
// @Accept       json
// @Produce      json
// @Param        id   path  int                                 true  "改期单ID"
// @Param        req  body  contract.StaffRescheduleAuditReq    true  "审批结论"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /wechat/staff/reschedule/audit/{id} [post]
func (h *Controller) RescheduleAudit(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req contract.StaffRescheduleAuditReq
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
// @Summary      退款审核
// @Description  **端差异实现**：员工端契约是 {"approve": bool}，PC 是 {"approved": *bool} 且必填；改任一侧都是破坏性变更。
// @Description  approve=false 表示驳回，布尔零值合法，故不使用 required 校验。
// @Tags         员工端·退款
// @Accept       json
// @Produce      json
// @Param        id   path  int                              true  "退款单ID"
// @Param        req  body  object{approve=bool,remark=string} true  "审核结论（approve=false 即驳回）"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /wechat/staff/refund/audit/{id} [post]
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
// @Summary      日程列表
// @Description  员工端日程（按日期区间查询档期占用）；复用管理端 ListCalendar。
// @Tags         员工端·日程
// @Accept       json
// @Produce      json
// @Param        req  body  object{start_date=string,end_date=string,photographer_id=int}  true  "查询条件（日期格式 2006-01-02）"
// @Success      200  {object}  response.Body{data=[]model.CalendarBlock}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /wechat/staff/schedule/list [post]
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
// @Summary      生成线索 AI 简报
// @Description  依据线索信息重建需求摘要（已确认项 + 待追问项），**覆盖旧数据**。
// @Tags         员工端·线索简报
// @Accept       json
// @Produce      json
// @Param        lead_id  path  int  true  "线索ID"
// @Success      200  {object}  response.Body{data=[]model.LeadBriefItem}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /wechat/staff/brief/generate/{lead_id} [post]
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
// @Summary      简报项列表
// @Tags         员工端·线索简报
// @Accept       json
// @Produce      json
// @Param        lead_id  path  int  true  "线索ID"
// @Success      200  {object}  response.Body{data=[]model.LeadBriefItem}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /wechat/staff/brief/list/{lead_id} [post]
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
// @Summary      发送追问
// @Description  把简报项作为追问消息发送给客户。
// @Tags         员工端·线索简报
// @Accept       json
// @Produce      json
// @Param        id  path  int  true  "简报项ID"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /wechat/staff/brief/send/{id} [post]
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
// @Summary      确认简报项
// @Description  人工确认/修正 AI 提取出的简报项取值（AI 提取结果需人工复核后生效）。
// @Tags         员工端·线索简报
// @Accept       json
// @Produce      json
// @Param        id   path  int                             true  "简报项ID"
// @Param        req  body  contract.StaffBriefConfirmReq    true  "确认值"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /wechat/staff/brief/confirm/{id} [post]
func (h *Controller) BriefConfirm(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req contract.StaffBriefConfirmReq
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
// 档期模板 / 评价 / 设置 / 设备
// （定制需求已上提至公共路由表，与 PC 共用 Handler，见 routes/customer.go）
// ---------------------------------------------------------------------

// ReviewList 评价列表
// @Summary      评价列表
// @Description  分页查询客户评价，可按最低星级过滤。
// @Tags         员工端·评价
// @Accept       json
// @Produce      json
// @Param        req  body  object{page=int,page_size=int,min_rating=int}  true  "查询条件（min_rating 0=不限）"
// @Success      200  {object}  response.Body{data=response.Page{list=[]model.OrderReview,total=int,page=int,page_size=int}}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /wechat/staff/review/list [post]
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
// @Summary      回复评价
// @Tags         员工端·评价
// @Accept       json
// @Produce      json
// @Param        id   path  int                            true  "评价ID"
// @Param        req  body  contract.StaffReviewReplyReq    true  "回复内容"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /wechat/staff/review/reply/{id} [post]
func (h *Controller) ReviewReply(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req contract.StaffReviewReplyReq
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
// @Summary      登录设备列表
// @Description  当前账号的登录设备（个人中心 → 设备管理；操作对象是登录者本人，免权限点）。
// @Tags         员工端·个人中心
// @Accept       json
// @Produce      json
// @Success      200  {object}  response.Body{data=[]model.UserDevice}
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /wechat/staff/device/list [post]
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
// @Summary      踢出登录设备
// @Description  强制下线某台登录设备（免权限点：操作对象是登录者本人设备）。
// @Tags         员工端·个人中心
// @Accept       json
// @Produce      json
// @Param        id  path  int  true  "设备ID"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /wechat/staff/device/remove/{id} [post]
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
// @Summary      今日待跟进
// @Description  返回到期/逾期且未成交未流失的线索（客户档案页今日待跟进）。跟进对象是线索。
// @Tags         员工端·客户档案
// @Accept       json
// @Produce      json
// @Param        req  body  object{limit=int}  true  "返回条数上限（0=默认）"
// @Success      200  {object}  response.Body{data=[]model.Lead}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /wechat/staff/customer/today-follow [post]
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
// @Summary      修改客户手机号
// @Description  为客户换绑手机号（含格式校验与租户内占用校验）。
// @Tags         员工端·客户档案
// @Accept       json
// @Produce      json
// @Param        req  body  contract.StaffCustomerMobileReq  true  "客户ID与新手机号"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /wechat/staff/customer/mobile [post]
func (h *Controller) CustomerMobileUpdate(c *gin.Context) {
	op := middleware.GetOperator(c)
	var req contract.StaffCustomerMobileReq
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
// @Summary      修图反馈列表
// @Description  客户修图反馈的待办列表（分页）。
// @Tags         员工端·反馈整理
// @Accept       json
// @Produce      json
// @Param        req  body  object{page=int,page_size=int,status=int}  true  "查询条件（status 1-待处理 2-已处理，0=全部）"
// @Success      200  {object}  response.Body{data=response.Page{list=[]repository.FeedbackListItem,total=int,page=int,page_size=int}}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /wechat/staff/delivery/feedback/list [post]
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
// @Summary      处理反馈
// @Description  标记反馈已处理并记录处理备注。⚠️ :item_id 是**交付文件 ID**。
// @Tags         员工端·反馈整理
// @Accept       json
// @Produce      json
// @Param        item_id  path  int                              true  "交付文件ID"
// @Param        req      body  contract.StaffFeedbackHandleReq   true  "处理备注"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /wechat/staff/delivery/feedback/handle/{item_id} [post]
func (h *Controller) FeedbackHandle(c *gin.Context) {
	op := middleware.GetOperator(c)
	itemID, err := bind.PathID(c, "item_id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req contract.StaffFeedbackHandleReq
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

// ---------------------------------------------------------------------
// 交付：发送最终确认（员工端独有）
// ---------------------------------------------------------------------

// DeliverySendFinal 发送最终确认（:id 为**交付单 ID**，与 delivery/confirm 同口径）。
// PC 交付工作台没有这个动作，属移动端独有，故不进公共路由表。
// @Summary      发送最终确认
// @Description  向客户发送最终确认。⚠️ :id 为**交付单 ID**（与 /delivery/confirm 同口径）。
// @Description  PC 交付工作台无此动作，属移动端独有能力，故不进公共路由表。
// @Tags         员工端·交付
// @Accept       json
// @Produce      json
// @Param        id  path  int  true  "交付单ID"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /wechat/staff/delivery/send-final/{id} [post]
func (h *Controller) DeliverySendFinal(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.SendFinalToCustomer(c.Request.Context(), op, id); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

// ---------------------------------------------------------------------
// 账号自助：换绑手机号（免权限点，与 /user/* 同属"操作本人账号"）
// ---------------------------------------------------------------------

// StaffMobileCode 发送换绑验证码（发往**当前绑定手机号**，scene=change_mobile）
// @Summary      发送换绑验证码
// @Description  验证码发往**当前绑定的手机号**，场景 scene=change_mobile（与登录场景隔离）。免权限点。
// @Tags         员工端·个人中心
// @Accept       json
// @Produce      json
// @Success      200  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /wechat/staff/user/mobile-code [post]
func (h *Controller) StaffMobileCode(c *gin.Context) {
	op := middleware.GetOperator(c)
	if err := h.Svc.SendStaffMobileCode(c.Request.Context(), op); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

// StaffChangeMobile 校验验证码并换绑本人手机号（body: {code, new_mobile}）
// @Summary      换绑手机号
// @Description  校验验证码后换绑本人手机号。免权限点（操作对象是登录者本人账号）。
// @Tags         员工端·个人中心
// @Accept       json
// @Produce      json
// @Param        req  body  contract.StaffChangeMobileReq  true  "验证码与新手机号"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /wechat/staff/user/change-mobile [post]
func (h *Controller) StaffChangeMobile(c *gin.Context) {
	op := middleware.GetOperator(c)
	var req contract.StaffChangeMobileReq
	if err := bind.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.ChangeStaffMobile(c.Request.Context(), op, req.Code, req.NewMobile); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

// ---------------------------------------------------------------------
// 意见反馈（免权限点：提交人本人）
// ---------------------------------------------------------------------

// FeedbackSubmit 提交意见反馈（body: contract.FeedbackSubmitReq）
// @Summary      提交意见反馈
// @Description  员工提交意见反馈，流转在管理后台处理。免权限点（提交人本人）。
// @Tags         员工端·个人中心
// @Accept       json
// @Produce      json
// @Param        req  body  contract.FeedbackSubmitReq  true  "反馈内容"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /wechat/staff/feedback/submit [post]
func (h *Controller) FeedbackSubmit(c *gin.Context) {
	op := middleware.GetOperator(c)
	var req contract.FeedbackSubmitReq
	if err := bind.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.SubmitFeedback(c.Request.Context(), op, req); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}
