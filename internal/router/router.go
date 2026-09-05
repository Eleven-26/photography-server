package router

import (
	"github.com/gin-gonic/gin"

	"photography-server/internal/config"
	"photography-server/internal/middleware"
	"photography-server/internal/presentation/controller"
	"photography-server/internal/service"
)

// New 构建 gin 引擎并按客户端分组注册 RPC 风格路由
// 客户端分组：pc-管理后台 miniapp-小程序管理后台 app-APP h5-移动端
func New(cfg *config.Config, svc *service.Service) *gin.Engine {
	if cfg.App.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	engine := gin.New()
	engine.Use(middleware.CORS(), middleware.Recovery(), middleware.RequestLog())
	// SkyWalking 链路追踪（OTel → otel-collector → SkyWalking OAP）：未启用时返回 nil，请求路径零影响
	if tm := middleware.SkyWalkingTrace(cfg.SkyWalking.Service); tm != nil {
		engine.Use(tm)
		// 把 entry span 的 trace_id 回写响应头 X-Trace-Id，便于日志/UI 检索；需注册在 otelgin 之后
		engine.Use(middleware.TraceID())
	}

	ctl := controller.New(svc, cfg)
	middleware.Init(cfg)
	mw := middleware.Get()

	// 静态资源：上传文件
	engine.Static("/uploads", svc.UploadDir)

	//api := engine.Group("/api")
	api := engine.Group("")
	api.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"code": 0, "msg": "ok", "data": "photography-server running"})
	})

	// 公共接口：登录（四个客户端统一走 /auth/login）
	api.POST("/auth/login", ctl.Login)

	// 各客户端业务路由分组（RPC 风格：/client/module/action/:id）
	pc := api.Group("", mw.Auth(), mw.OperationLog())
	miniapp := api.Group("/miniapp", mw.Auth(), mw.OperationLog())
	app := api.Group("/app", mw.Auth(), mw.OperationLog())
	h5 := api.Group("/h5", mw.Auth(), mw.OperationLog())

	registerCommon(pc, ctl)
	registerCommon(miniapp, ctl)
	registerCommon(app, ctl)
	registerCommon(h5, ctl)

	// 调试路由：仅非 release（dev/test）环境注册，生产环境不暴露基础设施操作能力
	if cfg.App.Mode != "release" {
		registerDebug(api, ctl)
	}

	return engine
}

// registerCommon 注册所有客户端共用的业务路由
func registerCommon(g *gin.RouterGroup, ctl *controller.Controller) {
	// 用户与权限
	u := g.Group("/user")
	u.POST("/profile", ctl.Profile)
	u.POST("/change-password", ctl.ChangePassword)
	u.POST("/logout", ctl.Logout)
	u.POST("/list", ctl.UserList)
	u.POST("/create", ctl.UserCreate)
	u.POST("/update/:id", ctl.UserUpdate)
	u.POST("/delete/:id", ctl.UserDelete)
	u.POST("/reset-password/:id", ctl.UserResetPassword)

	r := g.Group("/role")
	r.POST("/list", ctl.RoleList)
	r.POST("/create", ctl.RoleCreate)
	r.POST("/update/:id", ctl.RoleUpdate)
	r.POST("/delete/:id", ctl.RoleDelete)

	s := g.Group("/store")
	s.POST("/list", ctl.StoreList)
	s.POST("/create", ctl.StoreCreate)
	s.POST("/update/:id", ctl.StoreUpdate)
	s.POST("/delete/:id", ctl.StoreDelete)

	// 客户
	cu := g.Group("/customer")
	cu.POST("/list", ctl.CustomerList)
	cu.POST("/detail/:id", ctl.CustomerDetail)
	cu.POST("/create", ctl.CustomerCreate)
	cu.POST("/update/:id", ctl.CustomerUpdate)
	cu.POST("/delete/:id", ctl.CustomerDelete)
	cu.POST("/stats", ctl.CustomerStats)
	cu.POST("/orders/:id", ctl.CustomerOrders)

	// 线索
	ld := g.Group("/lead")
	ld.POST("/list", ctl.LeadList)
	ld.POST("/detail/:id", ctl.LeadDetail)
	ld.POST("/create", ctl.LeadCreate)
	ld.POST("/update/:id", ctl.LeadUpdate)
	ld.POST("/delete/:id", ctl.LeadDelete)
	ld.POST("/follow/:id", ctl.LeadFollow)
	ld.POST("/convert/:id", ctl.LeadConvert)

	// 报价单
	qt := g.Group("/quote")
	qt.POST("/create/:lead_id", ctl.QuoteCreate)
	qt.POST("/list/:lead_id", ctl.QuoteList)
	qt.POST("/status/:id", ctl.QuoteStatus)

	// 套餐
	pk := g.Group("/package")
	pk.POST("/list", ctl.PackageList)
	pk.POST("/detail/:id", ctl.PackageDetail)
	pk.POST("/create", ctl.PackageCreate)
	pk.POST("/update/:id", ctl.PackageUpdate)
	pk.POST("/status/:id", ctl.PackageStatus)
	pk.POST("/delete/:id", ctl.PackageDelete)

	// 订单
	od := g.Group("/order")
	od.POST("/create", ctl.OrderCreate)
	od.POST("/list", ctl.OrderList)
	od.POST("/detail/:id", ctl.OrderDetail)
	od.POST("/update/:id", ctl.OrderUpdate)
	od.POST("/status/:id", ctl.OrderStatus)
	od.POST("/cancel/:id", ctl.OrderCancel)
	od.POST("/logs/:id", ctl.OrderLogs)

	// 收款
	pm := g.Group("/payment")
	pm.POST("/create/:order_id", ctl.PaymentCreate)
	pm.POST("/list/:order_id", ctl.PaymentList)
	pm.POST("/confirm/:id", ctl.PaymentConfirm)
	pm.POST("/delete/:id", ctl.PaymentDelete)

	// 退款
	rf := g.Group("/refund")
	rf.POST("/apply/:order_id", ctl.RefundApply)
	rf.POST("/list/:order_id", ctl.RefundList)
	rf.POST("/audit/:id", ctl.RefundAudit)

	// 交付
	dv := g.Group("/delivery")
	dv.POST("/detail/:id", ctl.DeliveryDetail)
	dv.POST("/items/:id", ctl.DeliveryItems)
	dv.POST("/upload-samples/:id", ctl.DeliveryUploadSamples)
	dv.POST("/select/:id", ctl.DeliverySelect)
	dv.POST("/upload-retouched/:id", ctl.DeliveryUploadRetouched)
	dv.POST("/confirm/:id", ctl.DeliveryConfirm)

	// 作品集
	wk := g.Group("/asset")
	wk.POST("/list", ctl.AssetList)
	wk.POST("/detail/:id", ctl.AssetDetail)
	wk.POST("/create", ctl.AssetCreate)
	wk.POST("/update/:id", ctl.AssetUpdate)
	wk.POST("/delete/:id", ctl.AssetDelete)

	// 档期
	cal := g.Group("/calendar")
	cal.POST("/list", ctl.CalendarList)
	cal.POST("/lock", ctl.CalendarLock)
	cal.POST("/cancel/:id", ctl.CalendarCancel)

	// 财务
	fn := g.Group("/finance")
	fn.POST("/summary", ctl.FinanceSummary)
	fn.POST("/payments", ctl.FinancePayments)
	fn.POST("/refunds", ctl.FinanceRefunds)

	// 工作台
	dash := g.Group("/dashboard")
	dash.POST("/overview", ctl.DashboardOverview)

	// 通知
	nt := g.Group("/notification")
	nt.POST("/list", ctl.NotificationList)
	nt.POST("/unread-count", ctl.NotificationUnreadCount)
	nt.POST("/read/:id", ctl.NotificationRead)
	nt.POST("/read-all", ctl.NotificationReadAll)

	// 设置
	st := g.Group("/settings")
	st.POST("/workspace", ctl.Workspace)
	st.POST("/company/update", ctl.CompanyUpdate)
	st.POST("/payment-method/list", ctl.PaymentMethodList)
	st.POST("/payment-method/create", ctl.PaymentMethodCreate)
	st.POST("/payment-method/update/:id", ctl.PaymentMethodUpdate)
	st.POST("/payment-method/delete/:id", ctl.PaymentMethodDelete)
	st.POST("/operation-log/list", ctl.OperationLogList)

	// 上传
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
	t.POST("/skywalking/status", ctl.SkyWalkingStatus)
	t.POST("/skywalking/trace", ctl.SkyWalkingTrace)

	t.POST("/test", ctl.Test)
}
