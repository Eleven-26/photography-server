package router

import (
	"github.com/gin-gonic/gin"

	"photography-server/internal/app"
	"photography-server/internal/config"
	"photography-server/internal/domain"
	"photography-server/internal/middleware"
	"photography-server/internal/pkg/params"
	"photography-server/internal/presentation/controller"
	"photography-server/internal/presentation/h5"
	"photography-server/internal/presentation/wechat"
	"photography-server/internal/service"
)

// New 构建 gin 引擎并按客户端分组注册 RPC 风格路由
// 客户端分组：pc-管理后台 miniapp-小程序管理后台 wechat-客户小程序 h5-客户H5
// 端入口分开：管理端接口仅注册在 pc/miniapp；微信小程序为移动端统一入口（当前无独立 App），
// 客户区挂 /wechat（客户认证）、员工区挂 /wechat/staff（员工认证）；客户 H5 挂 /h5（客户认证）。
// mw / a 由组合根（main）构造后注入（#40），路由层不再触碰基础设施单例。
func New(cfg *config.Config, svc *service.Service, mw *middleware.Middlewares, a *app.App) *gin.Engine {
	if cfg.App.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	engine := gin.New()
	engine.Use(middleware.CORS(cfg.App.CORSOrigins), middleware.Recovery(), middleware.RequestLog())
	// 请求参数统一从 JSON body 取（含分页 page/page_size、keyword、status 等）：
	// 预解析 body 存入 context 并回填，供 params.Str/Int/Int64 读取；非 JSON 请求原样放行
	engine.Use(params.Middleware())
	// Jaeger 链路通道（OTel → Jaeger，复用 OTel 埋点）：未启用时返回 nil，请求路径零影响
	if tm := mw.JaegerTrace(); tm != nil {
		engine.Use(tm)
		// 把 entry span 的 trace_id 回写响应头 X-Trace-Id，便于日志/UI 检索；需注册在 otelgin 之后
		engine.Use(middleware.TraceID())
		// 把请求参数（query/JSON body）追加到 entry span 属性；需在 otelgin 之后、业务处理器之前
		engine.Use(middleware.TraceParams())
	} else {
		// 非 Jaeger/OTel 通道：SkyWalking-go agent 版（注入构建）的 entry span 由 agent 在 gin 外层自动创建，
		// TraceID 中间件从 agent 上下文取 native trace_id 回写响应头；纯本地开发（无任何追踪）时为 no-op。
		engine.Use(middleware.TraceID())
	}

	ctl := controller.New(svc, cfg, a)

	// 静态资源：上传文件（鉴权下载，禁止匿名枚举客户样片/成片/凭证）
	// 访问需携带有效登录令牌（员工或客户均可）；文件名服务端生成不可枚举（见 service/upload.go）
	uploads := engine.Group("/uploads", mw.AssetAuth())
	uploads.Static("/", svc.UploadDir)

	//api := engine.Group("/api")
	api := engine.Group("")
	api.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"code": 0, "msg": "ok", "data": "photography-server running"})
	})

	// 公共接口：登录（四个客户端统一走 /auth/login）
	api.POST("/auth/login", ctl.Login)

	// 管理端分组（员工认证）：pc-管理后台 miniapp-小程序管理后台
	pc := api.Group("", mw.Auth(), mw.OperationLog())
	miniapp := api.Group("/miniapp", mw.Auth(), mw.OperationLog())
	registerCommon(pc, ctl, mw)
	registerCommon(miniapp, ctl, mw)

	// ---- 客户 H5（客户验证码登录 + CustomerAuth）----
	h5Ctl := h5.New(svc, cfg)
	h5Pub := api.Group("/h5")
	h5Ctl.RegisterPublic(h5Pub)
	h5Auth := api.Group("/h5", mw.CustomerAuth())
	h5Ctl.RegisterAuthed(h5Auth)

	// ---- 微信小程序（移动端统一入口）----
	// 客户区（客户验证码登录 + CustomerAuth）
	wcCtl := wechat.New(svc, cfg)
	wcPub := api.Group("/wechat")
	wcCtl.RegisterPublic(wcPub)
	wcAuth := api.Group("/wechat", mw.CustomerAuth())
	wcCtl.RegisterAuthed(wcAuth)
	// 员工区（员工验证码登录 + StaffAuth）：摄影师/助理用小程序处理订单、日程、线索
	wcStaffPub := api.Group("/wechat/staff")
	wcCtl.RegisterStaffPublic(wcStaffPub)
	wcStaffAuth := api.Group("/wechat/staff", mw.StaffAuth(), mw.OperationLog())
	wcCtl.RegisterStaffAuthed(wcStaffAuth, mw)

	// 调试路由（Redis/NATS/ES/Mongo 读写删除 + 配置密文生成，无业务鉴权）：
	// 以 profile（服务端启动参数/APP_PROFILE，可信）白名单为准——只有 dev/test/docker.dev 注册；
	// 生产（prod）无论 app.mode 是否误配为 debug/空 都不暴露（旧实现按 mode!="release" 黑名单判断，
	// mode 为空即等于非 release → 生产误配会全量暴露，是 P0 隐患）。
	if debugProfile(cfg.App.Profile) {
		registerDebug(api, ctl)
	}

	return engine
}

// debugProfile 判断当前 profile 是否允许注册调试路由
func debugProfile(profile string) bool {
	switch profile {
	case "dev", "test", "docker.dev":
		return true
	}
	return false
}

// registerCommon 注册所有客户端共用的业务路由（PC 管理后台与小程序管理后台共用挂载点，权限点挂一次两端生效）。
//
// mw 仅用于在路由上追加权限点判定（mw.Perm(...)）；分组级认证由调用方的
// pc / miniapp 分组统一挂载，此处不重复认证。
//
// 【免挂权限点的路由（自助类 / 通用能力）】——这是**有意为之**，不是遗漏：
//   - /user/profile、/user/change-password、/user/logout：操作对象是登录者本人账号，
//     不属于角色能力边界，任何登录员工都必须可用（否则改密码都要管理员授权，属设计缺陷）。
//   - /upload/file：通用文件上传能力，被收款凭证、作品、交付文件等多条链路共用；
//     挂 asset:upload 会连带拦掉销售上传收款凭证（sales 无 asset:upload），属误伤。
//     上传内容的安全性由各业务接口自身的归属校验与文件类型白名单保证。
//
// 除此之外的每条业务路由都必须挂权限点——权限点的**唯一权威来源**是 domain.Perm 常量，
// 与 .doc/角色权限管理实施方案-2026-09-10.md §九 的角色矩阵对应。
func registerCommon(g *gin.RouterGroup, ctl *controller.Controller, mw *middleware.Middlewares) {
	// 用户与权限（profile / change-password / logout 为自助类，豁免权限点）
	u := g.Group("/user")
	u.POST("/profile", ctl.Profile)
	u.POST("/change-password", ctl.ChangePassword)
	u.POST("/logout", ctl.Logout)
	u.POST("/list", mw.Perm(domain.PermUserView), ctl.UserList)
	u.POST("/create", mw.Perm(domain.PermUserCreate), ctl.UserCreate)
	u.POST("/update/:id", mw.Perm(domain.PermUserUpdate), ctl.UserUpdate)
	u.POST("/delete/:id", mw.Perm(domain.PermUserDelete), ctl.UserDelete)
	u.POST("/reset-password/:id", mw.Perm(domain.PermUserResetPwd), ctl.UserResetPassword)

	r := g.Group("/role")
	r.POST("/list", mw.Perm(domain.PermRoleView), ctl.RoleList)
	r.POST("/create", mw.Perm(domain.PermRoleCreate), ctl.RoleCreate)
	r.POST("/update/:id", mw.Perm(domain.PermRoleUpdate), ctl.RoleUpdate)
	r.POST("/delete/:id", mw.Perm(domain.PermRoleDelete), ctl.RoleDelete)
	// 角色权限（RBAC）：配置入口是权限体系的"钥匙"，随接口一同挂载权限点，
	// 避免 B1-1 上线到 B1-2 挂载之间出现可被任意登录员工改权限的安全空窗。
	r.POST("/catalog", mw.Perm(domain.PermRoleView), ctl.RoleCatalog)
	r.POST("/permissions/:id", mw.Perm(domain.PermRoleView), ctl.RolePerms)
	r.POST("/grant/:id", mw.Perm(domain.PermRoleGrant), ctl.RoleGrant)

	s := g.Group("/store")
	s.POST("/list", mw.Perm(domain.PermStoreView), ctl.StoreList)
	s.POST("/create", mw.Perm(domain.PermStoreCreate), ctl.StoreCreate)
	s.POST("/update/:id", mw.Perm(domain.PermStoreUpdate), ctl.StoreUpdate)
	s.POST("/delete/:id", mw.Perm(domain.PermStoreDelete), ctl.StoreDelete)

	// 客户（stats / orders 为客户的只读派生视图，与 list/detail 同权限）
	cu := g.Group("/customer")
	cu.POST("/list", mw.Perm(domain.PermCustomerView), ctl.CustomerList)
	cu.POST("/detail/:id", mw.Perm(domain.PermCustomerView), ctl.CustomerDetail)
	cu.POST("/create", mw.Perm(domain.PermCustomerCreate), ctl.CustomerCreate)
	cu.POST("/update/:id", mw.Perm(domain.PermCustomerUpdate), ctl.CustomerUpdate)
	cu.POST("/delete/:id", mw.Perm(domain.PermCustomerDelete), ctl.CustomerDelete)
	cu.POST("/stats", mw.Perm(domain.PermCustomerView), ctl.CustomerStats)
	cu.POST("/orders/:id", mw.Perm(domain.PermCustomerView), ctl.CustomerOrders)

	// 线索（沟通记录与 AI 简报读接口归 lead:view，写入归 lead:update；
	// 变更归属人属"分配"语义，路由级无法区分，由 service 层追加 lead:assign 校验）
	ld := g.Group("/lead")
	ld.POST("/list", mw.Perm(domain.PermLeadView), ctl.LeadList)
	ld.POST("/detail/:id", mw.Perm(domain.PermLeadView), ctl.LeadDetail)
	ld.POST("/create", mw.Perm(domain.PermLeadCreate), ctl.LeadCreate)
	ld.POST("/update/:id", mw.Perm(domain.PermLeadUpdate), ctl.LeadUpdate)
	ld.POST("/delete/:id", mw.Perm(domain.PermLeadDelete), ctl.LeadDelete)
	ld.POST("/follow/:id", mw.Perm(domain.PermLeadUpdate), ctl.LeadFollow)
	ld.POST("/convert/:id", mw.Perm(domain.PermLeadConvert), ctl.LeadConvert)
	ld.POST("/messages/:id", mw.Perm(domain.PermLeadView), ctl.LeadMessages)
	ld.POST("/message/send/:id", mw.Perm(domain.PermLeadUpdate), ctl.LeadMessageSend)
	ld.POST("/brief/list/:lead_id", mw.Perm(domain.PermLeadView), ctl.LeadBriefList)
	ld.POST("/brief/generate/:lead_id", mw.Perm(domain.PermLeadUpdate), ctl.LeadBriefGenerate)

	// 报价单（status 为接受/拒绝/成交/撤回的状态流转，归 quote:update；
	// 报价无独立审批接口，quote:audit 已随 2026-09-11 权限点清理移除）
	qt := g.Group("/quote")
	qt.POST("/create/:lead_id", mw.Perm(domain.PermQuoteCreate), ctl.QuoteCreate)
	qt.POST("/list/:lead_id", mw.Perm(domain.PermQuoteView), ctl.QuoteList)
	qt.POST("/status/:id", mw.Perm(domain.PermQuoteUpdate), ctl.QuoteStatus)

	// 套餐（status 即上下架）
	pk := g.Group("/package")
	pk.POST("/list", mw.Perm(domain.PermPackageView), ctl.PackageList)
	pk.POST("/detail/:id", mw.Perm(domain.PermPackageView), ctl.PackageDetail)
	pk.POST("/create", mw.Perm(domain.PermPackageCreate), ctl.PackageCreate)
	pk.POST("/update/:id", mw.Perm(domain.PermPackageUpdate), ctl.PackageUpdate)
	pk.POST("/status/:id", mw.Perm(domain.PermPackagePublish), ctl.PackageStatus)
	pk.POST("/delete/:id", mw.Perm(domain.PermPackageDelete), ctl.PackageDelete)

	// 订单（金额无独立修改接口——改价经加项同事务重算，走 order:update）
	od := g.Group("/order")
	od.POST("/create", mw.Perm(domain.PermOrderCreate), ctl.OrderCreate)
	od.POST("/list", mw.Perm(domain.PermOrderView), ctl.OrderList)
	od.POST("/detail/:id", mw.Perm(domain.PermOrderView), ctl.OrderDetail)
	od.POST("/update/:id", mw.Perm(domain.PermOrderUpdate), ctl.OrderUpdate)
	od.POST("/status/:id", mw.Perm(domain.PermOrderStatus), ctl.OrderStatus)
	od.POST("/cancel/:id", mw.Perm(domain.PermOrderCancel), ctl.OrderCancel)
	od.POST("/logs/:id", mw.Perm(domain.PermOrderView), ctl.OrderLogs)

	// 订单加项（妆造/时效/服务/精修）：增删改同事务重算订单金额，归订单编辑权限
	od.POST("/addon/list/:order_id", mw.Perm(domain.PermOrderView), ctl.OrderAddonList)
	od.POST("/addon/create/:order_id", mw.Perm(domain.PermOrderUpdate), ctl.OrderAddonCreate)
	od.POST("/addon/update/:id", mw.Perm(domain.PermOrderUpdate), ctl.OrderAddonUpdate)
	od.POST("/addon/delete/:id", mw.Perm(domain.PermOrderUpdate), ctl.OrderAddonDelete)

	// 改期：PC 不直接改订单拍摄日期（会漏掉档期锁重排），统一走改期单链路
	// apply = 发起改期（销售/店长），audit = 审批改期（店长），二者权限点分离
	od.POST("/reschedule/list/:order_id", mw.Perm(domain.PermOrderView), ctl.OrderRescheduleList)
	od.POST("/reschedule/apply/:order_id", mw.Perm(domain.PermOrderReschedule), ctl.OrderRescheduleApply)
	od.POST("/reschedule/audit/:id", mw.Perm(domain.PermOrderRescheduleAudit), ctl.OrderRescheduleAudit)

	// 收款（登记与核验到账分离：登记不改变资金确认状态，核验才是）
	pm := g.Group("/payment")
	pm.POST("/create/:order_id", mw.Perm(domain.PermPaymentCreate), ctl.PaymentCreate)
	pm.POST("/list/:order_id", mw.Perm(domain.PermPaymentView), ctl.PaymentList)
	pm.POST("/confirm/:id", mw.Perm(domain.PermPaymentConfirm), ctl.PaymentConfirm)
	pm.POST("/delete/:id", mw.Perm(domain.PermPaymentDelete), ctl.PaymentDelete)

	// 退款（发起与审批分离：发起者不得自审）
	rf := g.Group("/refund")
	rf.POST("/apply/:order_id", mw.Perm(domain.PermRefundCreate), ctl.RefundApply)
	rf.POST("/list/:order_id", mw.Perm(domain.PermRefundView), ctl.RefundList)
	rf.POST("/audit/:id", mw.Perm(domain.PermRefundAudit), ctl.RefundAudit)

	// 交付（读接口归 view，上传/选片/确认等推进动作归 update）
	dv := g.Group("/delivery")
	dv.POST("/list", mw.Perm(domain.PermDeliveryView), ctl.DeliveryList)
	dv.POST("/create/:order_id", mw.Perm(domain.PermDeliveryCreate), ctl.DeliveryCreate)
	dv.POST("/remind/:id", mw.Perm(domain.PermDeliveryUpdate), ctl.DeliveryRemind)
	dv.POST("/detail/:id", mw.Perm(domain.PermDeliveryView), ctl.DeliveryDetail)
	dv.POST("/items/:id", mw.Perm(domain.PermDeliveryView), ctl.DeliveryItems)
	dv.POST("/upload-samples/:id", mw.Perm(domain.PermDeliveryUpdate), ctl.DeliveryUploadSamples)
	dv.POST("/select/:id", mw.Perm(domain.PermDeliveryUpdate), ctl.DeliverySelect)
	dv.POST("/upload-retouched/:id", mw.Perm(domain.PermDeliveryUpdate), ctl.DeliveryUploadRetouched)
	dv.POST("/confirm/:id", mw.Perm(domain.PermDeliveryUpdate), ctl.DeliveryConfirm)

	// 作品集（status 接口控制可见性/精选，属"发布审核"动作 → asset:audit；
	// 故摄影师可上传/编辑自己的作品，但无权决定是否公开——需店长审核）
	wk := g.Group("/asset")
	wk.POST("/list", mw.Perm(domain.PermAssetView), ctl.AssetList)
	wk.POST("/detail/:id", mw.Perm(domain.PermAssetView), ctl.AssetDetail)
	wk.POST("/create", mw.Perm(domain.PermAssetUpload), ctl.AssetCreate)
	wk.POST("/update/:id", mw.Perm(domain.PermAssetUpdate), ctl.AssetUpdate)
	wk.POST("/status/:id", mw.Perm(domain.PermAssetAudit), ctl.AssetStatus)
	wk.POST("/delete/:id", mw.Perm(domain.PermAssetDelete), ctl.AssetDelete)

	// 档期
	cal := g.Group("/calendar")
	cal.POST("/list", mw.Perm(domain.PermCalendarView), ctl.CalendarList)
	cal.POST("/lock", mw.Perm(domain.PermCalendarUpdate), ctl.CalendarLock)
	cal.POST("/cancel/:id", mw.Perm(domain.PermCalendarUpdate), ctl.CalendarCancel)
	// 档期规则（排班时段模板）
	cal.POST("/slot-template/list", mw.Perm(domain.PermCalendarView), ctl.SlotTemplateList)
	cal.POST("/slot-template/save", mw.Perm(domain.PermCalendarUpdate), ctl.SlotTemplateSave)
	cal.POST("/slot-template/save/:id", mw.Perm(domain.PermCalendarUpdate), ctl.SlotTemplateSave)
	cal.POST("/slot-template/delete/:id", mw.Perm(domain.PermCalendarUpdate), ctl.SlotTemplateDelete)

	// 财务（导出单独收敛：可见不等于可带走）
	fn := g.Group("/finance")
	fn.POST("/summary", mw.Perm(domain.PermFinanceView), ctl.FinanceSummary)
	fn.POST("/payments", mw.Perm(domain.PermFinanceView), ctl.FinancePayments)
	fn.POST("/refunds", mw.Perm(domain.PermFinanceView), ctl.FinanceRefunds)
	fn.POST("/export", mw.Perm(domain.PermFinanceExport), ctl.FinanceExport)

	// 工作台
	dash := g.Group("/dashboard")
	dash.POST("/overview", mw.Perm(domain.PermDashboardView), ctl.DashboardOverview)

	// 通知（读通知是每个员工的基础能力；该点控制"能否进入通知中心"）
	nt := g.Group("/notification")
	nt.POST("/list", mw.Perm(domain.PermNotificationView), ctl.NotificationList)
	nt.POST("/unread-count", mw.Perm(domain.PermNotificationView), ctl.NotificationUnreadCount)
	nt.POST("/read/:id", mw.Perm(domain.PermNotificationView), ctl.NotificationRead)
	nt.POST("/read-all", mw.Perm(domain.PermNotificationView), ctl.NotificationReadAll)

	// 设置（operation-log 归 log:view；收款方式维护归 settings:update）
	st := g.Group("/settings")
	st.POST("/workspace", mw.Perm(domain.PermSettingsView), ctl.Workspace)
	st.POST("/company/update", mw.Perm(domain.PermSettingsUpdate), ctl.CompanyUpdate)
	st.POST("/payment-method/list", mw.Perm(domain.PermSettingsView), ctl.PaymentMethodList)
	st.POST("/payment-method/create", mw.Perm(domain.PermSettingsUpdate), ctl.PaymentMethodCreate)
	st.POST("/payment-method/update/:id", mw.Perm(domain.PermSettingsUpdate), ctl.PaymentMethodUpdate)
	st.POST("/payment-method/delete/:id", mw.Perm(domain.PermSettingsUpdate), ctl.PaymentMethodDelete)
	st.POST("/operation-log/list", mw.Perm(domain.PermLogView), ctl.OperationLogList)
	st.POST("/studio/get", mw.Perm(domain.PermSettingsView), ctl.StudioGet)
	st.POST("/studio/update", mw.Perm(domain.PermSettingsUpdate), ctl.StudioUpdate)

	// 上传（通用文件上传能力，各业务链路共用 → 免挂权限点，见函数头说明）
	up := g.Group("/upload")
	up.POST("/file", ctl.UploadFile)
}

// registerDebug 注册基础设施调试路由（Redis / NATS / ES / Mongo 连通性与读写实验）。
// 仅由 New 在非 release 环境调用；这些处理器直连 infrastructure 单例（调试控制台），
// 不经过 service 层，也未挂业务鉴权——禁止在生产环境启用。
func registerDebug(g *gin.RouterGroup, ctl *controller.Controller) {
	t := g.Group("/test")
	t.POST("/redis/ping", ctl.RedisPing)
	t.POST("/redis/set", ctl.RedisSet)
	t.POST("/redis/get", ctl.RedisGet)
	t.POST("/redis/del", ctl.RedisDel)
	t.POST("/nats/status", ctl.NATSStatus)
	t.POST("/nats/pub", ctl.NATSPub)
	t.POST("/nats/pub-persistent", ctl.NATSPubPersistent)
	t.POST("/nats/pub-pull", ctl.NATSPubPull)
	t.POST("/nats/request", ctl.NATSRequest)
	t.POST("/es/status", ctl.ESStatus)
	t.POST("/es/index", ctl.ESIndex)
	t.POST("/es/search", ctl.ESSearch)
	t.POST("/es/list", ctl.ESList)
	t.POST("/es/delete", ctl.ESDelete)
	t.POST("/mongo/status", ctl.MongoStatus)
	t.POST("/mongo/insert", ctl.MongoInsert)
	t.POST("/mongo/insert-many", ctl.MongoInsertMany)
	t.POST("/mongo/find", ctl.MongoFind)
	t.POST("/mongo/find-one", ctl.MongoFindOne)
	t.POST("/mongo/update", ctl.MongoUpdate)
	t.POST("/mongo/delete", ctl.MongoDelete)
	t.POST("/mongo/delete-by-id", ctl.MongoDeleteByID)
	t.POST("/jaeger/status", ctl.JaegerStatus)
	t.POST("/jaeger/trace", ctl.JaegerTrace)
	// 配置密文生成（只加密不解密，未挂业务鉴权，仅非 release 注册）
	t.POST("/config/encrypt", ctl.ConfigEncrypt)

	t.POST("/test", ctl.Test)
}
