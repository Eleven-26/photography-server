package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"photography-server/internal/presentation/job"
	"syscall"
	"time"

	// SkyWalking Go agent（skywalking-go）：编译期无侵入注入探针（go build -toolexec="<agent>" -a）。
	// 无注入的普通构建（本地 go run / Jaeger 版）此 import 为零副作用占位；注入构建时 agent 自启，
	// 自动埋点 gin HTTP 入口与 gorm SQL，数据直连 OAP native gRPC(:11800)，Horizon「原生」模式可见。
	// 两通道各自独立、互不依赖，但勿同时开启（同一请求会双 span/双上报）：
	//   ① SkyWalking-go(native)：Dockerfile SW_AGENT_ENABLE=true 构建注入即启用，无运行时开关；
	//   ② OTel→Jaeger：不注入 agent，jaeger.enable=true 时启用 OTel exporter（下方 InitJaeger）。
	_ "github.com/apache/skywalking-go"

	"photography-server/internal/config"
	"photography-server/internal/infrastructure"
	"photography-server/internal/middleware"
	"photography-server/internal/pkg/logger"
	"photography-server/internal/presentation/mq"
	"photography-server/internal/router"
	"photography-server/internal/service"
)

func main() {
	var configPath string
	var profile string
	flag.StringVar(&configPath, "c", "config/config.yaml", "配置文件路径")
	flag.StringVar(&profile, "p", os.Getenv("APP_PROFILE"), "运行环境 dev|test|prod（默认 dev）")
	flag.Parse()

	if profile == "" {
		profile = "dev"
	}

	// Nacos 是唯一业务配置源：本地文件仅为 bootstrap（Nacos 连接信息），业务配置 100% 从 Nacos 拉取
	// （data_id 按 -p/APP_PROFILE 区分），拉取失败直接终止启动（fail-fast）；SDK 不可达时自动降级本地快照。
	cfg, err := config.LoadWithFetcher(configPath, profile, infrastructure.FetchConfig)
	if err != nil {
		panic(fmt.Sprintf("加载配置失败: %v", err))
	}
	logger.Init(cfg.Log.Level)
	logger.Infof("running profile: %s", cfg.App.Profile)

	// 业务时区（#16）：统一用配置的 timezone 设置 time.Local——domain.ParseShootDate、
	// 各处 ParseInLocation(..., time.Local)、DSN loc=Local 全部依赖它，保证退款/改期档位
	// 与日期边界按业务时区计算，而不是容器 TZ 兜底。
	if cfg.App.Timezone != "" {
		if loc, err := time.LoadLocation(cfg.App.Timezone); err != nil {
			logger.Warnf("timezone %q 无效（%v），使用系统时区", cfg.App.Timezone, err)
		} else {
			time.Local = loc
			logger.Infof("timezone set to %s", cfg.App.Timezone)
		}
	}

	// Jaeger 链路通道（OTel SDK → Jaeger，复用 OTel 埋点）；未启用时跳过，Jaeger 不可达不影响启动。
	// 注意：必须先于 InitMySQL —— GORM 的 OTel 插件在安装时捕获全局 TracerProvider，
	// 顺序颠倒会导致 SQL span 走 noop provider，永远不产生数据。
	if err := infrastructure.InitJaeger(&cfg.Jaeger); err != nil {
		logger.Warnf("jaeger not available, skipping: %v", err)
	}

	if err := infrastructure.InitMySQL(cfg); err != nil {
		panic(fmt.Sprintf("连接数据库失败: %v", err))
	}
	if err := infrastructure.Ping(); err != nil {
		panic(fmt.Sprintf("数据库不可用: %v", err))
	}
	logger.Infof("ping database ok")

	bootstrap(cfg.App.Profile) // #25：prod 环境内部禁用（见 bootstrap.go）

	// Redis 提升为硬依赖（#12）：验证码登录、JWT 登出黑名单、认证画像缓存全部依赖 Redis，
	// 旧实现失败仅 warn 后继续启动，会让服务"看似正常但客户端全部无法登录/登出"。
	// 与 MySQL 同级 fail-fast；NATS/ES/Mongo/XXL 仍为可选（失败降级不影响主流程）。
	if err := infrastructure.InitRedis(&cfg.Redis); err != nil {
		panic(fmt.Sprintf("连接 Redis 失败（验证码登录/令牌吊销/认证缓存为硬依赖）: %v", err))
	}

	if err := infrastructure.InitNATS(&cfg.NATS); err != nil {
		logger.Warnf("nats not available, skipping: %v", err)
	}

	if err := infrastructure.InitES(&cfg.ES); err != nil {
		logger.Warnf("elasticsearch not available, skipping: %v", err)
	}

	if err := infrastructure.InitMongoDB(&cfg.Mongo); err != nil {
		logger.Warnf("mongodb not available, skipping: %v", err)
	}

	if err := infrastructure.InitXxlJob(cfg); err != nil {
		logger.Warnf("xxl-job not available, skipping: %v", err)
	}
	if executor := infrastructure.XxlExecutor(); executor != nil {
		job.Register(executor)
		infrastructure.RunXxlJob()
	}

	// 启动 NATS 消费者（实例外提：优雅退出时先 Stop 再关连接）
	var consumer *mq.Consumer
	if nc := infrastructure.NATS(); nc != nil {
		consumer = mq.New(nc)
		consumer.Start()
	}

	middleware.Init(cfg)
	svc := service.New(cfg.Upload.Dir, cfg.JWT.Secret, cfg.JWT.Issuer)
	engine := router.New(cfg, svc)

	// HTTP Server 超时（#26）：防 Slowloris 类慢速攻击用极少量连接长期占满 goroutine/文件描述符。
	// ReadHeaderTimeout 必设（读请求头自身无业务超时兜底）；WriteTimeout 覆盖整请求写响应。
	srv := &http.Server{
		Handler:           engine,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	// 先 Listen 成功再启动 serve，保证后续 Nacos 注册的实例一定可服务
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.App.Port))
	if err != nil {
		panic(fmt.Sprintf("监听端口失败: %v", err))
	}
	go func() {
		logger.Infof("photography-server listening on %s", ln.Addr())
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Errorf("server error: %v", err)
			os.Exit(1)
		}
	}()

	// Nacos：服务注册（配置链路已 fail-fast，注册失败只告警不阻断服务——注册是服务发现增强）。
	// 临时实例：SDK 自动心跳，进程退出自动摘除（shutdown 时另有主动反注册）。
	if _, err := infrastructure.RegisterService(cfg.App.Name, cfg.App.Port, map[string]string{"profile": cfg.App.Profile}); err != nil {
		logger.Warnf("nacos 服务注册失败: %v", err)
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Infof("shutting down...")

	// ---- 优雅关闭（#27）：顺序 = 停止入口 → 停消费者 → 停 HTTP → flush 链路 → 关基础设施 ----
	// 1. 先停 NATS 消费者（防止关闭期间仍在消费/回包）
	if consumer != nil {
		consumer.Stop()
	}
	// 2. HTTP 停止接收新请求并等待在途请求完成（5s 上限）
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	if err := srv.Shutdown(ctx); err != nil {
		logger.Errorf("shutdown error: %v", err)
	}
	// 3. 独立短 ctx flush Jaeger span（不复用上面可能已耗尽的 5s 上下文，避免末批链路数据丢失）
	jctx, jcancel := context.WithTimeout(context.Background(), 3*time.Second)
	infrastructure.CloseJaeger(jctx)
	jcancel()
	cancel()
	// 4. 基础设施连接与后台执行器
	infrastructure.CloseXxlJob()
	infrastructure.CloseNATS()
	infrastructure.CloseRedis()
	infrastructure.CloseES()
	infrastructure.MongoDisconnect(context.Background())
	infrastructure.CloseNacos()
	infrastructure.CloseMySQL()
	logger.Infof("shutdown complete")
}
