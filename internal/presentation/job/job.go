package job

import (
	"context"
	"fmt"
	"time"

	xxl "github.com/xxl-job/xxl-job-executor-go"

	"photography-server/internal/pkg/logger"
)

// Register 注册所有定时任务到 XXL-JOB 执行器
func Register(executor xxl.Executor) {
	executor.RegTask("job.test", TestJob)
	executor.RegTask("job.health_check", HealthCheckJob)
	logger.Infof("xxl-job tasks registered")
}

// TestJob 测试任务 —— 在 XXL-JOB 管理后台新建任务，JobHandler 填 job.test 即可触发
func TestJob(cxt context.Context, param *xxl.RunReq) string {
	logger.Infof("[TestJob] start, param: %s", param.ExecutorParams)
	start := time.Now()

	// TODO: 在此编写具体业务逻辑
	time.Sleep(200 * time.Millisecond)

	logger.Infof("[TestJob] done, cost: %s", time.Since(start))
	return fmt.Sprintf("测试任务执行成功，耗时 %s", time.Since(start))
}

// HealthCheckJob 健康检查任务 —— JobHandler: job.health_check
func HealthCheckJob(cxt context.Context, param *xxl.RunReq) string {
	logger.Infof("[HealthCheckJob] start")
	logger.Infof("[HealthCheckJob] done")
	return "ok"
}
