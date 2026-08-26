package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"photography-server/internal/config"
	"photography-server/internal/database"
	"photography-server/internal/pkg/logger"
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

	db, err := database.New(cfg)
	if err != nil {
		panic(fmt.Sprintf("连接数据库失败: %v", err))
	}
	if err := database.Ping(db); err != nil {
		panic(fmt.Sprintf("数据库不可用: %v", err))
	}
	logger.Infof("ping database ok")

	bootstrap(db)

	svc := service.New(db, cfg.Upload.Dir)
	engine := router.New(cfg, db, svc)

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
}
