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

	// nacos.enable=true 时：本地配置退化为 bootstrap，业务配置从 Nacos 拉取（远程优先），
	// 拉取失败直接返回错误终止启动（fail-fast）；enable=false 时 fetcher 不触发，与本地加载行为一致。
	cfg, err := config.LoadWithFetcher(configPath, profile, infrastructure.FetchConfig)
	if err != nil {
		panic(fmt.Sprintf("加载配置失败: %v", err))
	}
	logger.Init(cfg.Log.Level)
	logger.Infof("running profile: %s", cfg.App.Profile)

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

	bootstrap()

	if err := infrastructure.InitRedis(&cfg.Redis); err != nil {
		logger.Warnf("redis not available, skipping: %v", err)
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

	// 启动 NATS 消费者
	if nc := infrastructure.NATS(); nc != nil {
		consumer := mq.New(nc)
		consumer.Start()
	}

	middleware.Init(cfg)
	svc := service.New(cfg.Upload.Dir, cfg.JWT.Secret, cfg.JWT.Issuer)
	engine := router.New(cfg, svc)

	srv := &http.Server{Handler: engine}

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

	// Nacos：服务注册（nacos.enable=true 时生效；失败只告警不影响服务）。
	// 临时实例：SDK 自动心跳，进程退出自动摘除（shutdown 时另有主动反注册）。
	if cfg.Nacos.Enable {
		if _, err := infrastructure.RegisterService(cfg.App.Name, cfg.App.Port, map[string]string{"profile": cfg.App.Profile}); err != nil {
			logger.Warnf("nacos 服务注册失败: %v", err)
		}
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Infof("shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Errorf("shutdown error: %v", err)
	}
	infrastructure.CloseJaeger(ctx)
	infrastructure.CloseNacos()
}
