package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"photography-server/internal/presentation/job"
	"syscall"

	xxl "github.com/xxl-job/xxl-job-executor-go"

	"photography-server/internal/config"
	"photography-server/internal/pkg/logger"
)

// 这个可以单独启动调试，上线时无需启动，因为在主程序main有启动
func main() {
	var configPath string
	var profile string
	flag.StringVar(&configPath, "c", "config/config.yaml", "配置文件路径")
	flag.StringVar(&profile, "p", "dev", "运行环境 dev|test|prod")
	flag.Parse()

	cfg, err := config.Load(configPath, profile)
	if err != nil {
		panic(fmt.Sprintf("加载配置失败: %v", err))
	}
	logger.Init(cfg.Log.Level)

	if !cfg.XxlJob.Enable {
		logger.Infof("xxl-job is disabled, set xxljob.enable=true to start")
		os.Exit(0)
	}

	executor := xxl.NewExecutor(
		xxl.ServerAddr(cfg.XxlJob.ServerAddr),
		xxl.AccessToken(cfg.XxlJob.AccessToken),
		xxl.ExecutorIp(cfg.XxlJob.ExecutorIp),
		xxl.ExecutorPort(cfg.XxlJob.ExecutorPort),
		xxl.RegistryKey(cfg.XxlJob.RegistryKey),
	)
	executor.Init()

	job.Register(executor)

	go func() {
		if err := executor.Run(); err != nil {
			logger.Errorf("xxl-job executor error: %v", err)
			os.Exit(1)
		}
	}()

	logger.Infof("xxl-job executor started, waiting for tasks...")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Infof("xxl-job executor shutting down...")
}
