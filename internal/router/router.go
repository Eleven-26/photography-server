package router

import (
	"path/filepath"

	"github.com/gin-gonic/gin"

	"photography-server/internal/app"
	"photography-server/internal/config"
	"photography-server/internal/middleware"
	"photography-server/internal/pkg/params"
	"photography-server/internal/presentation/controller"
	"photography-server/internal/presentation/endpoint"
	"photography-server/internal/presentation/h5"
	"photography-server/internal/presentation/routes"
	"photography-server/internal/presentation/wechat"
	"photography-server/internal/service"
)

// New 构建 gin 引擎并按客户端分组注册 RPC 风格路由
// 客户端分组：pc-管理后台 miniapp-小程序管理后台 wechat-客户小程序 h5-客户H5 wechat/staff-员工小程序
// 端入口分开：管理端接口仅注册在 pc/miniapp；微信小程序为移动端统一入口（当前无独立 App），
// 客户区挂 /wechat（客户认证）、员工区挂 /wechat/staff（员工认证）；客户 H5 挂 /h5（客户认证）。
// mw / a 由组合根（main）构造后注入（#40），路由层不再触碰基础设施单例。
//
// 注册方式（2026-09-12 路由整理后）：业务路由由 routes 包统一持有（端无关的唯一实现），
// 各端只声明暴露面（router/endpoints.go），此处遍历挂载。此前 registerCommon 硬绑
// *controller.Controller 且是包私有，wechat 包引用不到，员工端只能复制 24 条路由与 handler。
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
	// 端无关业务路由表：一处定义，各端按暴露面复用（同 Path = 同 handler 函数指针）
	commonRoutes := routes.Common(ctl)

	// 静态资源：上传文件（鉴权下载，禁止匿名枚举客户样片/成片/凭证）
	// 访问需携带有效登录令牌（员工或客户均可）；文件名服务端生成不可枚举（见 service/upload.go）
	uploads := engine.Group("/uploads", mw.AssetAuth())
	uploads.Static("/", svc.UploadDir)

	// 静态资源：公开媒体（作品集封面/图集）
	// 对外宣传物料，浏览者（H5 分享页）通常未登录，故**不挂鉴权**；
	// 与 /uploads 物理隔离（落盘在 <UploadDir>/media 子目录，见 service.UploadOptions.Public），
	// 订单样片/成片/付款凭证等隐私文件仍只在 /uploads 下、必须带令牌。
	// 上传侧由 /upload/file 的 public=1 显式指定，不存在"默认公开"路径。
	// 目录名与 URL 根路径由同一个常量推出（URL 根 = "/" + 目录名）：
	// 两处各写字面量一旦漂移，症状是"上传成功但图片 404"，排查成本高。
	engine.Static("/"+service.PublicMediaDir, filepath.Join(svc.UploadDir, service.PublicMediaDir))

	api := engine.Group("")
	api.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"code": 0, "msg": "ok", "data": "photography-server running"})
	})

	// 公共接口：管理端登录（pc / miniapp 两个管理后台共用 /auth/login，账号密码）
	api.POST("/auth/login", ctl.Login)

	// ---- 管理端（员工认证）：pc-管理后台 miniapp-小程序管理后台 ----
	// 同一份路由表挂两个前缀，这正是端复用的既有约定（员工端此前偏离了它）。
	endpoint.Mount(api, pcEndpoint(mw), commonRoutes, mw)
	endpoint.Mount(api, miniappEndpoint(mw), commonRoutes, mw)

	// ---- 客户区（H5 与小程序客户区共用同一份路由表）----
	// 路由声明在 routes 包：ClientPublic（免登录）/ ClientAuthed（分组挂 CustomerAuth）。
	// 两个端引用的是**同一份 []Route** —— 同一 Route 实例、同一 Handler 函数指针，
	// 仅挂载分组不同。两端口径由构造保证，不再依赖"记得两边一起改"（routes_test 亦据此断言）。
	// 此前客户区是唯一命令式内联注册（g.POST 48 条）的端，游离在声明式表与跨端护栏之外。
	h5Ctl := h5.New(svc, cfg)
	clientPublic := routes.ClientPublic(h5Ctl)
	clientAuthed := routes.ClientAuthed(h5Ctl)

	// 客户 H5：/h5（公开）+ /h5 + CustomerAuth（客户验证码登录，令牌 → 客户上下文）
	endpoint.MountTable(api.Group("/h5"), clientPublic)
	endpoint.MountTable(api.Group("/h5", mw.CustomerAuth()), clientAuthed)

	// 微信小程序（移动端统一入口）：客户区 /wechat 与 H5 同源（同一份表，仅前缀不同）
	endpoint.MountTable(api.Group("/wechat"), clientPublic)
	endpoint.MountTable(api.Group("/wechat", mw.CustomerAuth()), clientAuthed)

	// ---- 微信小程序员工区（员工账号密码登录 + StaffAuth）----
	// 摄影师/助理用小程序处理订单、日程、线索 AI 简报。含三类路由（见 endpoints.go）：
	//   Include     复用 PC 管理端同一 Handler；
	//   Extra       异路径别名 / 端差异实现 / 移动端独有；
	//   PublicExtra 登录出口 /wechat/staff/auth/*（登录发生在拿到令牌之前，不挂 StaffAuth）。
	wcCtl := wechat.New(svc, cfg)
	endpoint.Mount(api, staffEndpoint(wcCtl, ctl, mw), commonRoutes, mw)

	// 调试路由（配置密文生成 + [仅 debug 构建] 基础设施读写实验，均无业务鉴权）：
	// 两级把关——
	//   ① 编译期（debug 构建标签）：Redis/NATS/ES/Mongo/Jaeger 调试接口只在 -tags debug 产物中
	//      编译（见 debug_on.go），默认构建（含生产镜像）在二进制层面即不含这些代码；
	//   ② 运行期（profile 白名单 dev/test/docker.dev）：prod 无论 app.mode 是否误配为 debug/空
	//      都不注册（旧实现按 mode!="release" 黑名单判断，mode 为空即等于非 release → 生产误配会全量暴露，是 P0 隐患）。
	registerDebugRoutes(api, ctl, cfg.App.Profile)

	return engine
}
