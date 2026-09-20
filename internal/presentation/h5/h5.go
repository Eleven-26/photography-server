// Package h5 客户 H5 端接口（客户预约全链路：浏览套餐 → 提交预约 → 支付定金 →
// 选片 → 确认成片 → 评价）。公开接口无需登录，业务接口经 CustomerAuth 注入客户上下文。
//
// 路由注册自 2026-09-16 起**不在本包**：客户区 48 条路由声明于
// presentation/routes/client.go 的 ClientPublic / ClientAuthed，由 router 按端与鉴权分组挂载，
// 且 H5 与小程序客户区引用**同一份 []Route**（同一 Handler 函数指针）。
// 本文件只承载 handler 实现与端内辅助（slugFrom / staffFrom / requireCompany）。
package h5

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"photography-server/internal/config"
	"photography-server/internal/contract"
	"photography-server/internal/middleware"
	"photography-server/internal/pkg/errs"
	"photography-server/internal/pkg/params"
	"photography-server/internal/pkg/response"
	"photography-server/internal/presentation/bind"
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

// 路由注册已迁出本文件（2026-09-16）：
// 客户区 48 条路由现声明在 presentation/routes/client.go 的 ClientPublic / ClientAuthed，
// 由 router 按端与鉴权分组挂载 —— H5 与小程序客户区引用的是同一份 []Route
// （同一 Handler 函数指针），两端口径由构造保证，护栏测试亦可覆盖。
// 本文件从此只保留 handler 实现与辅助函数。

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

// staffFrom 提取分享链接携带的员工账号 ID：X-Staff-Id 头 → body staff_id → query ?staff_id=
// （与 slugFrom 同款优先级）。链接形如 https://host/?slug=xxx&staff_id=12，由员工端
// 「我的预约主页」分享出去（见 contract.NewStaffStudioSettingResp）。
// 用途：客户从谁的链接进来下单，订单就归到该员工名下（biz_order.photographer_id），
// 员工端「仅本人」数据范围据此能查到自己的客户单。
// 缺失或非法一律返回 0 —— 视为「非分享进入」，不报错，订单由工作室后续指派。
func staffFrom(c *gin.Context) int64 {
	candidates := []string{c.GetHeader("X-Staff-Id"), params.Str(c, "staff_id"), c.Query("staff_id")}
	for _, raw := range candidates {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		if id, err := strconv.ParseInt(raw, 10, 64); err == nil && id > 0 {
			return id
		}
	}
	return 0
}

// requireCompany 按 slug 反查并校验租户（数据库不存在/未配置 slug 时返回业务错误）
func (h *Controller) requireCompany(c *gin.Context) (int64, error) {
	slug := slugFrom(c)
	if slug == "" {
		return 0, errs.BadRequest(errs.ErrSlugRequired)
	}
	companyID, err := h.Svc.ResolveCompanyBySlug(c.Request.Context(), slug)
	if err != nil {
		return 0, errs.Internal("")
	}
	if companyID <= 0 {
		return 0, errs.BadRequest(errs.ErrHomepageNotConfigured)
	}
	return companyID, nil
}

// ---------------------------------------------------------------------
// 公开接口
// ---------------------------------------------------------------------

// SmsCode 发送登录验证码
// @Summary      发送登录验证码
// @Description  短信发往指定手机号，场景固定为 login；短信通道未接入时验证码只打服务端日志。
// @Tags         客户区·登录
// @Accept       json
// @Produce      json
// @Param        req  body  object{mobile=string}  true  "手机号"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Router       /h5/auth/sms-code [post]
func (h *Controller) SmsCode(c *gin.Context) {
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

// Login 客户手机号验证码登录（未注册自动建档），openid 为小程序场景透传。
// 租户定位：body slug（可选）→ query slug / X-Slug 头 → 服务端反查 company_id（#29）
// @Summary      客户登录
// @Description  手机号 + 短信验证码登录；**未注册的手机号自动建档客户**（手机号为租户内唯一键）。
// @Description  租户定位：body.slug（也可放 query slug / X-Slug 头）；服务端按 slug 反查 company_id，**不接受客户端直传 company_id**。
// @Description  校验规则：dev / docker.dev 允许免验证码登录（短信通道未接入），test / prod 及未知 profile 强制校验验证码。
// @Description  登录成功返回客户令牌，后续请求以 "Bearer <token>" 放入 Authorization 头。
// @Tags         客户区·登录
// @Accept       json
// @Produce      json
// @Param        req  body  object{slug=string,mobile=string,code=string,openid=string}  true  "登录信息（mobile 必填；openid 为小程序场景透传）"
// @Success      200  {object}  response.Body{data=object{token=string,customer=model.Customer}}
// @Failure      400  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Router       /h5/auth/login [post]
func (h *Controller) Login(c *gin.Context) {
	var req struct {
		Slug   string `json:"slug"` // 预约主页短链标识（也可放 query/头）
		Mobile string `json:"mobile" binding:"required"`
		Code   string `json:"code"` // 短信验证码；开发环境免验证码时可为空（见 loginRequireSmsCode）
		OpenID string `json:"openid"`
	}
	if err := bind.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	slug := req.Slug
	if slug == "" {
		slug = slugFrom(c)
	}
	if slug == "" {
		response.Fail(c, errs.BadRequest(errs.ErrSlugRequired))
		return
	}
	companyID, err := h.Svc.ResolveCompanyBySlug(c.Request.Context(), slug)
	if err != nil {
		response.Fail(c, errs.Internal(""))
		return
	}
	if companyID <= 0 {
		response.Fail(c, errs.BadRequest(errs.ErrHomepageNotConfigured))
		return
	}
	customer, token, err := h.Svc.CustomerSmsLogin(c.Request.Context(), companyID, req.Mobile, req.Code, req.OpenID, h.loginRequireSmsCode())
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{
		"token":    token,
		"customer": customer,
	})
}

// loginRequireSmsCode 登录是否必须短信验证码。
//
// 仅本地开发环境（dev / docker.dev）放行「免验证码登录」：短信通道尚未接入，
// 验证码只打到服务端日志，联调时逐个去日志捞不现实。
//
// 为什么不做成配置项：免验证码 == 「知道手机号即可登录该客户账号」，而客户账号能读
// 自己的订单、交付样片、评价与个人资料。这种开关一旦可配，就有被误配到生产的风险
// （且误配后没有任何报错，直到有人发现能拿别人手机号登录），故写成**白名单判断**：
// 只有显式跑在 dev / docker.dev 才放开，test / prod 及一切未知 profile 一律强制校验。
// 若将来确有其它环境需要，改这个 switch —— 不要在配置里开一个自由开关。
func (h *Controller) loginRequireSmsCode() bool {
	switch h.Cfg.App.Profile {
	case "dev", "docker.dev":
		return false
	}
	return true
}

// PackageList 套餐列表（已上架）
// @Summary      套餐列表
// @Description  预约主页套餐列表，**只返回已上架套餐**。
// @Description  租户定位：body.slug（也可放 query slug / X-Slug 头）；服务端按 slug 反查 company_id，**不接受客户端直传 company_id**。
// @Tags         客户区·套餐
// @Accept       json
// @Produce      json
// @Param        req  body  object{page=int,page_size=int,category=string}  true  "查询条件（slug 见说明）"
// @Success      200  {object}  response.Body{data=response.Page{list=[]model.Package,total=int,page=int,page_size=int}}
// @Failure      400  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Router       /h5/package/list [post]
func (h *Controller) PackageList(c *gin.Context) {
	companyID, err := h.requireCompany(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	page, pageSize := bind.Pager(c)
	list, total, err := h.Svc.ClientPackages(c.Request.Context(), companyID, page, pageSize, params.Str(c, "category"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.PageOK(c, list, total, page, pageSize)
}

// PackageDetail 套餐详情
// @Summary      套餐详情
// @Description  套餐详情（预约主页点开套餐卡片）。
// @Description  租户定位：body.slug（也可放 query slug / X-Slug 头）；服务端按 slug 反查 company_id，**不接受客户端直传 company_id**。
// @Tags         客户区·套餐
// @Accept       json
// @Produce      json
// @Param        id  path  int  true  "套餐ID"
// @Success      200  {object}  response.Body{data=model.Package}
// @Failure      400  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Router       /h5/package/detail/{id} [post]
func (h *Controller) PackageDetail(c *gin.Context) {
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
	pkg, err := h.Svc.ClientPackageDetail(c.Request.Context(), companyID, id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, pkg)
}

// StudioInfo 工作室预约主页信息
// @Summary      工作室信息
// @Description  预约主页展示用的工作室信息（名称 / 简介 / 联系方式 / 分享链接等）。
// @Description  租户定位：body.slug（也可放 query slug / X-Slug 头）；服务端按 slug 反查 company_id，**不接受客户端直传 company_id**。
// @Tags         客户区·工作室
// @Accept       json
// @Produce      json
// @Success      200  {object}  response.Body{data=model.StudioSetting}
// @Failure      400  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Router       /h5/studio/info [post]
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
// @Summary      可约时段
// @Description  查询指定日期的可约时段；photographer_id 可选，用于只看某位摄影师的空闲时段。
// @Description  租户定位：body.slug（也可放 query slug / X-Slug 头）；服务端按 slug 反查 company_id，**不接受客户端直传 company_id**。
// @Tags         客户区·工作室
// @Accept       json
// @Produce      json
// @Param        req  body  object{date=string,photographer_id=int}  true  "查询条件（date 格式 2006-01-02；slug 见说明）"
// @Success      200  {object}  response.Body{data=[]contract.ClientSlot}
// @Failure      400  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Router       /h5/slot/list [post]
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

// CustomRequestSubmit 提交定制需求（游客/登录均可）。
// 摄影师归属（2026-09-15 补齐）：body.photographer_id（客户在 H5 定制需求页的显式选择）优先，
// 分享链接的 staff_id 兜底 —— 两者都缺则落门店/公共池，见 service.ClientSubmitCustomRequest。
// @Summary      提交定制需求
// @Description  提交定制需求，**游客与登录客户均可**。已登录取令牌内公司，游客按 slug 反查租户。
// @Description  摄影师归属：body.photographer_id（客户显式选择）优先，分享链接的 staff_id 兜底，两者都缺则落门店/公共池。
// @Description  ⚠️ 本接口会往 crm_customer 建档（手机号唯一键），公开可调，存在被刷数据的风险。
// @Tags         客户区·定制需求
// @Accept       json
// @Produce      json
// @Param        req  body  contract.ClientCustomRequestReq  true  "定制需求信息"
// @Success      200  {object}  response.Body{data=model.CustomRequest}
// @Failure      400  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Router       /h5/custom-request/submit [post]
func (h *Controller) CustomRequestSubmit(c *gin.Context) {
	var req contract.ClientCustomRequestReq
	if err := bind.BindJSON(c, &req); err != nil {
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
	m, err := h.Svc.ClientSubmitCustomRequest(c.Request.Context(), companyID, cu, req, staffFrom(c))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, m)
}

// ---------------------------------------------------------------------
// 登录后接口
// ---------------------------------------------------------------------

// BookingSubmit 提交预约单。
// 分享人归属：链接参数 staff_id（头/body/query 三选一，见 staffFrom）随预约一并落到订单，
// 客户从谁的预约主页进来下单，订单就算谁的。
// @Summary      提交预约单
// @Description  客户提交预约单。分享人归属：链接参数 staff_id 随预约落到订单（biz_order.photographer_id）。
// @Tags         客户区·订单
// @Accept       json
// @Produce      json
// @Param        req  body  contract.ClientBookingReq  true  "预约信息"
// @Success      200  {object}  response.Body{data=model.Order}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /h5/order/submit [post]
func (h *Controller) BookingSubmit(c *gin.Context) {
	cu := middleware.GetClientUser(c)
	var req contract.ClientBookingReq
	if err := bind.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	o, err := h.Svc.ClientSubmitBooking(c.Request.Context(), cu, req, staffFrom(c))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, o)
}

// BookingConfirm 确认预约单
// @Summary      确认预约单
// @Tags         客户区·订单
// @Accept       json
// @Produce      json
// @Param        id  path  int  true  "订单ID"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /h5/order/confirm/{id} [post]
func (h *Controller) BookingConfirm(c *gin.Context) {
	cu := middleware.GetClientUser(c)
	id, err := bind.PathID(c, "id")
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
// @Summary      取消预约单
// @Description  取消本人的预约单并记录原因；body 可为空（reason 可选）。
// @Tags         客户区·订单
// @Accept       json
// @Produce      json
// @Param        id   path  int                   true  "订单ID"
// @Param        req  body  object{reason=string} false  "取消原因（可省略）"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /h5/order/cancel/{id} [post]
func (h *Controller) BookingCancel(c *gin.Context) {
	cu := middleware.GetClientUser(c)
	id, err := bind.PathID(c, "id")
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
// @Summary      我的订单
// @Description  分页查询当前客户名下订单，可按状态过滤（归属由令牌内 customer_id 锁定）。
// @Tags         客户区·订单
// @Accept       json
// @Produce      json
// @Param        req  body  object{page=int,page_size=int,status=string}  true  "查询条件"
// @Success      200  {object}  response.Body{data=response.Page{list=[]model.Order,total=int,page=int,page_size=int}}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /h5/order/list [post]
func (h *Controller) OrderList(c *gin.Context) {
	cu := middleware.GetClientUser(c)
	page, pageSize := bind.Pager(c)
	list, total, err := h.Svc.ClientOrders(c.Request.Context(), cu, page, pageSize, params.Str(c, "status"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.PageOK(c, list, total, page, pageSize)
}

// OrderDetail 订单详情
// @Summary      订单详情
// @Tags         客户区·订单
// @Accept       json
// @Produce      json
// @Param        id  path  int  true  "订单ID"
// @Success      200  {object}  response.Body{data=contract.ClientOrderDetail}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /h5/order/detail/{id} [post]
func (h *Controller) OrderDetail(c *gin.Context) {
	cu := middleware.GetClientUser(c)
	id, err := bind.PathID(c, "id")
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

// OrderPrepRead 客户确认已读「拍前准备清单」（写 biz_order.prep_read_at，幂等）
// @Summary      拍前准备已读
// @Description  客户确认已读「拍前准备清单」，写 biz_order.prep_read_at；**幂等**。
// @Tags         客户区·订单
// @Accept       json
// @Produce      json
// @Param        id  path  int  true  "订单ID"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /h5/order/prep/read/{id} [post]
func (h *Controller) OrderPrepRead(c *gin.Context) {
	cu := middleware.GetClientUser(c)
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.ClientReadOrderPrep(c.Request.Context(), cu, id); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

// RescheduleApply 申请改期
// @Summary      申请改期
// @Description  客户发起改期申请（apply_source=2）。
// @Tags         客户区·改期
// @Accept       json
// @Produce      json
// @Param        order_id  path  int                         true  "订单ID"
// @Param        req       body  contract.ClientRescheduleReq true  "改期信息"
// @Success      200  {object}  response.Body{data=model.OrderReschedule}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /h5/reschedule/apply/{order_id} [post]
func (h *Controller) RescheduleApply(c *gin.Context) {
	cu := middleware.GetClientUser(c)
	orderID, err := bind.PathID(c, "order_id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req contract.ClientRescheduleReq
	if err := bind.BindJSON(c, &req); err != nil {
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
// @Summary      撤回改期申请
// @Tags         客户区·改期
// @Accept       json
// @Produce      json
// @Param        id  path  int  true  "改期单ID"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /h5/reschedule/cancel/{id} [post]
func (h *Controller) RescheduleCancel(c *gin.Context) {
	cu := middleware.GetClientUser(c)
	id, err := bind.PathID(c, "id")
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

// RescheduleList 我的订单改期单列表（改期进度页；不分页，逐单明细集合）
// @Summary      我的改期单列表
// @Description  返回某订单下的改期单（改期进度页；**不分页**，逐单明细集合）。
// @Tags         客户区·改期
// @Accept       json
// @Produce      json
// @Param        order_id  path  int  true  "订单ID"
// @Success      200  {object}  response.Body{data=[]model.OrderReschedule}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /h5/reschedule/list/{order_id} [post]
func (h *Controller) RescheduleList(c *gin.Context) {
	cu := middleware.GetClientUser(c)
	orderID, err := bind.PathID(c, "order_id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	list, err := h.Svc.ClientReschedules(c.Request.Context(), cu, orderID)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, list)
}

// RefundApply 申请退款
// @Summary      申请退款
// @Description  客户对订单发起退款申请；金额为空时按退款规则自动计算。
// @Tags         客户区·退款
// @Accept       json
// @Produce      json
// @Param        order_id  path  int                     true  "订单ID"
// @Param        req       body  contract.ClientRefundReq true  "退款请求"
// @Success      200  {object}  response.Body{data=model.OrderRefund}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /h5/refund/apply/{order_id} [post]
func (h *Controller) RefundApply(c *gin.Context) {
	cu := middleware.GetClientUser(c)
	orderID, err := bind.PathID(c, "order_id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req contract.ClientRefundReq
	if err := bind.BindJSON(c, &req); err != nil {
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

// RefundList 我的订单退款记录（退款进度页 C21；不分页，逐单明细集合）
// @Summary      我的退款记录
// @Description  返回某订单下的退款记录（退款进度页；**不分页**）。
// @Tags         客户区·退款
// @Accept       json
// @Produce      json
// @Param        order_id  path  int  true  "订单ID"
// @Success      200  {object}  response.Body{data=[]model.OrderRefund}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /h5/refund/list/{order_id} [post]
func (h *Controller) RefundList(c *gin.Context) {
	cu := middleware.GetClientUser(c)
	orderID, err := bind.PathID(c, "order_id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	list, err := h.Svc.ClientRefunds(c.Request.Context(), cu, orderID)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, list)
}

// RefundConfirm 客户确认收到退款（写 customer_confirm_at，与员工端审批闭环；幂等）
// @Summary      确认收到退款
// @Description  客户确认收到退款，写 customer_confirm_at，与员工端审批闭环；**幂等**。
// @Tags         客户区·退款
// @Accept       json
// @Produce      json
// @Param        id  path  int  true  "退款单ID"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /h5/refund/confirm/{id} [post]
func (h *Controller) RefundConfirm(c *gin.Context) {
	cu := middleware.GetClientUser(c)
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.ClientConfirmRefundReceived(c.Request.Context(), cu, id); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

// PaymentList 我的订单收款记录（支付页展示登记状态）
// @Summary      我的收款记录
// @Description  返回某订单的收款记录（支付页展示登记状态；**不分页**）。
// @Tags         客户区·收款
// @Accept       json
// @Produce      json
// @Param        order_id  path  int  true  "订单ID"
// @Success      200  {object}  response.Body{data=[]model.OrderPayment}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /h5/payment/list/{order_id} [post]
func (h *Controller) PaymentList(c *gin.Context) {
	cu := middleware.GetClientUser(c)
	orderID, err := bind.PathID(c, "order_id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	list, err := h.Svc.ClientPayments(c.Request.Context(), cu, orderID)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, list)
}

// PaymentMark 客户登记转账（「我已完成转账，通知摄影师」）。
// 资金不经平台：仅落 status=1 待核验记录，到账确认仍在员工端 /payment/confirm/:id。
// @Summary      登记转账
// @Description  客户登记「我已完成转账，通知摄影师」。资金不经平台：仅落一条待核验记录，到账确认仍在员工端 /payment/confirm/{id}。
// @Tags         客户区·收款
// @Accept       json
// @Produce      json
// @Param        req  body  contract.ClientPaymentMarkReq  true  "转账登记（含 order_id）"
// @Success      200  {object}  response.Body{data=model.OrderPayment}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /h5/pay/mark [post]
func (h *Controller) PaymentMark(c *gin.Context) {
	cu := middleware.GetClientUser(c)
	var req contract.ClientPaymentMarkReq
	if err := bind.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	p, err := h.Svc.ClientRegisterPayment(c.Request.Context(), cu, req.OrderID, req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, p)
}

// PaymentMethods 客户可见的收款方式（只出启用项，供支付页展示收款码/账号）
// @Summary      收款方式列表
// @Description  客户可见的收款方式，**只返回启用项**，供支付页展示收款码/账号。
// @Tags         客户区·收款
// @Accept       json
// @Produce      json
// @Success      200  {object}  response.Body{data=[]contract.ClientPaymentMethodResp}
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /h5/payment-method/list [post]
func (h *Controller) PaymentMethods(c *gin.Context) {
	cu := middleware.GetClientUser(c)
	list, err := h.Svc.ClientPaymentMethods(c.Request.Context(), cu)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, list)
}

// ReviewCreate 评价订单
// @Summary      评价订单
// @Tags         客户区·评价
// @Accept       json
// @Produce      json
// @Param        order_id  path  int                     true  "订单ID"
// @Param        req       body  contract.ClientReviewReq true  "评价内容"
// @Success      200  {object}  response.Body{data=model.OrderReview}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /h5/review/create/{order_id} [post]
func (h *Controller) ReviewCreate(c *gin.Context) {
	cu := middleware.GetClientUser(c)
	orderID, err := bind.PathID(c, "order_id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req contract.ClientReviewReq
	if err := bind.BindJSON(c, &req); err != nil {
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

// ReviewList 我的评价（客户中心 → 我的评价，只读；不分页的业务集合，同改期/退款列表口径）
// @Summary      我的评价
// @Description  客户中心 → 我的评价，只读；按令牌内 customer_id 锁定归属（**不分页**，同改期/退款列表口径）。
// @Tags         客户区·评价
// @Accept       json
// @Produce      json
// @Success      200  {object}  response.Body{data=[]repository.ReviewListItem}
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /h5/review/list [post]
func (h *Controller) ReviewList(c *gin.Context) {
	cu := middleware.GetClientUser(c)
	list, err := h.Svc.ClientReviews(c.Request.Context(), cu)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, list)
}

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

// DeliveryDetail 交付单与明细（选片页/成片页）。:id 为 **order_id**（与 PC 端同语义）。
// @Summary      交付单与明细
// @Description  选片页 / 成片页数据源。⚠️ 本接口的 :id 是 **order_id**（与 PC 端同语义）。
// @Tags         客户区·交付
// @Accept       json
// @Produce      json
// @Param        id  path  int  true  "订单ID"
// @Success      200  {object}  response.Body{data=object{delivery=model.Delivery,items=[]model.DeliveryItem}}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /h5/delivery/detail/{id} [post]
func (h *Controller) DeliveryDetail(c *gin.Context) {
	cu := middleware.GetClientUser(c)
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	d, items, err := h.Svc.ClientDeliveryDetail(c.Request.Context(), cu, id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{"delivery": d, "items": items})
}

// DeliveryItems 交付文件明细（按订单反查）。:id 为 **order_id**。
// 与 /delivery/detail/:id 数据同源，供「文件管理」tab 直接取列表；未建交付单返回空列表。
// @Summary      交付文件明细
// @Description  按订单反查交付文件明细，供「文件管理」tab 直接取列表；未建交付单返回空列表。⚠️ :id 是 **order_id**。
// @Tags         客户区·交付
// @Accept       json
// @Produce      json
// @Param        id  path  int  true  "订单ID"
// @Success      200  {object}  response.Body{data=[]model.DeliveryItem}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /h5/delivery/items/{id} [post]
func (h *Controller) DeliveryItems(c *gin.Context) {
	cu := middleware.GetClientUser(c)
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	_, items, err := h.Svc.ClientDeliveryDetail(c.Request.Context(), cu, id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, items)
}

// SelectPhotos 提交选片
// @Summary      提交选片
// @Description  ⚠️ 本接口的 :id 是**交付单 ID**。
// @Tags         客户区·交付
// @Accept       json
// @Produce      json
// @Param        id   path  int                      true  "交付单ID"
// @Param        req  body  object{item_ids=[]int64} true  "选中的文件 ID 列表"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /h5/delivery/select/{id} [post]
func (h *Controller) SelectPhotos(c *gin.Context) {
	cu := middleware.GetClientUser(c)
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req struct {
		ItemIDs []int64 `json:"item_ids" binding:"required"`
	}
	if err := bind.BindJSON(c, &req); err != nil {
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
// @Summary      确认加片费用
// @Description  ⚠️ 本接口的 :id 是**交付单 ID**。
// @Tags         客户区·交付
// @Accept       json
// @Produce      json
// @Param        id  path  int  true  "交付单ID"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /h5/delivery/confirm-extra/{id} [post]
func (h *Controller) ConfirmExtra(c *gin.Context) {
	cu := middleware.GetClientUser(c)
	id, err := bind.PathID(c, "id")
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
// @Summary      确认成片
// @Description  ⚠️ 本接口的 :id 是**交付单 ID**。
// @Tags         客户区·交付
// @Accept       json
// @Produce      json
// @Param        id  path  int  true  "交付单ID"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /h5/delivery/confirm/{id} [post]
func (h *Controller) ConfirmDelivery(c *gin.Context) {
	cu := middleware.GetClientUser(c)
	id, err := bind.PathID(c, "id")
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
// @Summary      提交精修反馈
// @Description  ⚠️ 本接口的 :item_id 是**交付文件 ID**。
// @Tags         客户区·交付
// @Accept       json
// @Produce      json
// @Param        item_id  path  int                       true  "交付文件ID"
// @Param        req      body  contract.ClientFeedbackReq true  "反馈内容"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /h5/delivery/feedback/{item_id} [post]
func (h *Controller) FeedbackSubmit(c *gin.Context) {
	cu := middleware.GetClientUser(c)
	itemID, err := bind.PathID(c, "item_id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req contract.ClientFeedbackReq
	if err := bind.BindJSON(c, &req); err != nil {
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
// @Summary      我的定制需求
// @Description  客户本人提交过的定制需求历史（分页）。
// @Tags         客户区·定制需求
// @Accept       json
// @Produce      json
// @Param        req  body  contract.PageReq  true  "分页参数（page / page_size）"
// @Success      200  {object}  response.Body{data=response.Page{list=[]model.CustomRequest,total=int,page=int,page_size=int}}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /h5/custom-request/list [post]
func (h *Controller) CustomRequestList(c *gin.Context) {
	cu := middleware.GetClientUser(c)
	page, pageSize := bind.Pager(c)
	list, total, err := h.Svc.ClientCustomRequests(c.Request.Context(), cu, page, pageSize)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.PageOK(c, list, total, page, pageSize)
}

// ---------------------------------------------------------------------
// 作品集（报告 H5，公开接口）
// ---------------------------------------------------------------------

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

// ---------------------------------------------------------------------
// 报价（报告 H1）
// ---------------------------------------------------------------------

// QuoteList 我的报价单列表（含明细字段，前端按 id 取单条即可）
// @Summary      我的报价单列表
// @Description  返回本人的报价单（含明细字段，前端按 id 取单条即可）。
// @Tags         客户区·报价
// @Accept       json
// @Produce      json
// @Success      200  {object}  response.Body{data=[]model.Quote}
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /h5/quote/list [post]
func (h *Controller) QuoteList(c *gin.Context) {
	cu := middleware.GetClientUser(c)
	list, err := h.Svc.ClientQuotes(c.Request.Context(), cu)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, list)
}

// QuoteDetail 单张报价详情（报价详情页，按 id 直取；归属校验含线索兜底）
// @Summary      报价详情
// @Description  单张报价详情，按 id 直取；归属校验含线索兜底。
// @Tags         客户区·报价
// @Accept       json
// @Produce      json
// @Param        id  path  int  true  "报价单ID"
// @Success      200  {object}  response.Body{data=model.Quote}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /h5/quote/detail/{id} [post]
func (h *Controller) QuoteDetail(c *gin.Context) {
	cu := middleware.GetClientUser(c)
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	q, err := h.Svc.ClientQuoteDetail(c.Request.Context(), cu, id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, q)
}

// QuoteAccept 接受报价
// @Summary      接受报价
// @Tags         客户区·报价
// @Accept       json
// @Produce      json
// @Param        id  path  int  true  "报价单ID"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /h5/quote/accept/{id} [post]
func (h *Controller) QuoteAccept(c *gin.Context) {
	cu := middleware.GetClientUser(c)
	id, err := bind.PathID(c, "id")
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
// @Summary      对报价提修改意见
// @Tags         客户区·报价
// @Accept       json
// @Produce      json
// @Param        id   path  int                          true  "报价单ID"
// @Param        req  body  contract.ClientQuoteModifyReq true  "修改意见"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /h5/quote/modify/{id} [post]
func (h *Controller) QuoteModify(c *gin.Context) {
	cu := middleware.GetClientUser(c)
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req contract.ClientQuoteModifyReq
	if err := bind.BindJSON(c, &req); err != nil {
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
// @Summary      改期单详情
// @Description  改期单详情，含调度费支付状态。
// @Tags         客户区·改期
// @Accept       json
// @Produce      json
// @Param        id  path  int  true  "改期单ID"
// @Success      200  {object}  response.Body{data=contract.ClientRescheduleDetailResp}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /h5/reschedule/detail/{id} [post]
func (h *Controller) RescheduleDetail(c *gin.Context) {
	cu := middleware.GetClientUser(c)
	id, err := bind.PathID(c, "id")
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
// @Summary      提交调度费凭证
// @Description  提交改期调度费的支付凭证（资金不经平台，走「上传凭证 → 工作室核验」）。
// @Tags         客户区·改期
// @Accept       json
// @Produce      json
// @Param        id   path  int                               true  "改期单ID"
// @Param        req  body  contract.ClientReschedulePayReq    true  "支付凭证"
// @Success      200  {object}  response.Body{data=model.OrderPayment}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /h5/reschedule/pay/{id} [post]
func (h *Controller) ReschedulePay(c *gin.Context) {
	cu := middleware.GetClientUser(c)
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req contract.ClientReschedulePayReq
	if err := bind.BindJSON(c, &req); err != nil {
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
// @Summary      加片费试算
// @Description  按已选张数试算加片费；**body 可为空**（缺省按当前已选张数试算）。⚠️ :id 是**交付单 ID**。
// @Tags         客户区·交付
// @Accept       json
// @Produce      json
// @Param        id   path  int                            true  "交付单ID"
// @Param        req  body  contract.ClientExtraQuoteReq    false  "试算参数（可省略）"
// @Success      200  {object}  response.Body{data=contract.ClientExtraQuoteResp}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /h5/delivery/extra-quote/{id} [post]
func (h *Controller) ExtraQuote(c *gin.Context) {
	cu := middleware.GetClientUser(c)
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req contract.ClientExtraQuoteReq
	_ = c.ShouldBindJSON(&req)
	q, err := h.Svc.ClientExtraQuote(c.Request.Context(), cu, id, req.SelectCount)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, q)
}

// OrderRequirementUpdate 客户修改拍摄需求（仅待定金/待拍摄，白名单字段）
// @Summary      修改拍摄需求
// @Description  客户修改拍摄需求，**仅待定金 / 待拍摄状态可改，且只接受白名单字段**。
// @Tags         客户区·订单
// @Accept       json
// @Produce      json
// @Param        id   path  int                                  true  "订单ID"
// @Param        req  body  contract.ClientOrderRequirementReq    true  "需求字段（白名单）"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /h5/order/requirement/update/{id} [post]
func (h *Controller) OrderRequirementUpdate(c *gin.Context) {
	cu := middleware.GetClientUser(c)
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req contract.ClientOrderRequirementReq
	if err := bind.BindJSON(c, &req); err != nil {
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
// @Summary      我的通知列表
// @Description  客户本人的站内通知（分页）；unread=1 只看未读。
// @Tags         客户区·通知
// @Accept       json
// @Produce      json
// @Param        req  body  object{page=int,page_size=int,unread=string}  true  "查询条件（unread=1 只看未读）"
// @Success      200  {object}  response.Body{data=response.Page{list=[]model.SysNotification,total=int,page=int,page_size=int}}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /h5/notification/list [post]
func (h *Controller) NotificationList(c *gin.Context) {
	cu := middleware.GetClientUser(c)
	page, pageSize := bind.Pager(c)
	list, total, err := h.Svc.ListClientNotifications(c.Request.Context(), cu, page, pageSize, params.Str(c, "unread") == "1")
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.PageOK(c, list, total, page, pageSize)
}

// NotificationUnreadCount 未读通知数（铃铛红点）
// @Summary      未读通知数
// @Description  铃铛红点用。
// @Tags         客户区·通知
// @Accept       json
// @Produce      json
// @Success      200  {object}  response.Body{data=object{count=int}}
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /h5/notification/unread-count [post]
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
// @Summary      标记通知已读
// @Tags         客户区·通知
// @Accept       json
// @Produce      json
// @Param        id  path  int  true  "通知ID"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /h5/notification/read/{id} [post]
func (h *Controller) NotificationRead(c *gin.Context) {
	cu := middleware.GetClientUser(c)
	id, err := bind.PathID(c, "id")
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
// @Summary      全部标记已读
// @Tags         客户区·通知
// @Accept       json
// @Produce      json
// @Success      200  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /h5/notification/read-all [post]
func (h *Controller) NotificationReadAll(c *gin.Context) {
	cu := middleware.GetClientUser(c)
	if err := h.Svc.MarkAllClientNotificationsRead(c.Request.Context(), cu); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}
