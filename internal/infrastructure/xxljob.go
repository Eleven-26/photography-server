package infrastructure

import (
	"sync"

	xxl "github.com/xxl-job/xxl-job-executor-go"

	"photography-server/internal/config"
	"photography-server/internal/pkg/logger"
)

var (
	xxlExecutor xxl.Executor
	xxlOnce     sync.Once
	xxlErr      error
)

// InitXxlJob 初始化 XXL-JOB 执行器单例
func InitXxlJob(c *config.Config) error {
	xxlOnce.Do(func() {
		if !c.XxlJob.Enable {
			logger.Infof("xxl-job disabled, skipping")
			return
		}
		executor := xxl.NewExecutor(
			xxl.ServerAddr(c.XxlJob.ServerAddr),
			xxl.AccessToken(c.XxlJob.AccessToken),
			xxl.ExecutorIp(c.XxlJob.ExecutorIp),
			xxl.ExecutorPort(c.XxlJob.ExecutorPort),
			xxl.RegistryKey(c.XxlJob.RegistryKey),
		)
		executor.Init()
		xxlExecutor = executor
		logger.Infof("xxl-job executor initialized, server: %s, key: %s", c.XxlJob.ServerAddr, c.XxlJob.RegistryKey)
	})
	return xxlErr
}

// XxlExecutor 获取 XXL-JOB 执行器单例
func XxlExecutor() xxl.Executor {
	return xxlExecutor
}

// RunXxlJob 启动 XXL-JOB 执行器（在 goroutine 中运行）
func RunXxlJob() {
	if xxlExecutor == nil {
		return
	}
	go func() {
		if err := xxlExecutor.Run(); err != nil {
			logger.Errorf("xxl-job executor error: %v", err)
		}
	}()
	logger.Infof("xxl-job executor started")
}
