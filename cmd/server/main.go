package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"photography-server/internal/presentation/job"
	"syscall"
	"time"

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

	cfg, err := config.Load(configPath, profile)
	if err != nil {
		panic(fmt.Sprintf("加载配置失败: %v", err))
	}
	logger.Init(cfg.Log.Level)
	logger.Infof("running profile: %s", cfg.App.Profile)

	// SkyWalking 链路追踪：OTel SDK → otel-collector → SkyWalking OAP；未启用时跳过，collector 不可达不影响启动。
	// 注意：必须先于 InitMySQL —— GORM 的 OTel 插件在安装时捕获全局 TracerProvider，
	// 顺序颠倒会导致 SQL span 走 noop provider，永远不产生数据。
	if err := infrastructure.InitSkyWalking(&cfg.SkyWalking); err != nil {
		logger.Warnf("skywalking not available, skipping: %v", err)
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
	svc := service.New(cfg.Upload.Dir)
	engine := router.New(cfg, svc)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.App.Port),
		Handler: engine,
	}

	go func() {
		logger.Infof("photography-server listening on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Errorf("server error: %v", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Infof("shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Errorf("shutdown error: %v", err)
	}
	infrastructure.CloseSkyWalking(ctx)
}
