package job

import (
	"context"
	"fmt"
	"time"

	xxl "github.com/xxl-job/xxl-job-executor-go"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"photography-server/internal/pkg/logger"
)

// Register 注册所有定时任务到 XXL-JOB 执行器。
// tracer 由组合根注入（#40；链路未启用时为 nil，traced 退化为直调，零开销）。
func Register(executor xxl.Executor, tracer trace.Tracer) {
	executor.RegTask("job.test", traced(tracer, "job.test", TestJob))
	executor.RegTask("job.health_check", traced(tracer, "job.health_check", HealthCheckJob))
	logger.Infof("xxl-job tasks registered")
}

// traced 为 xxl 任务包一层链路追踪（Jaeger/OTel 通道）：
// go-client 的任务 ctx 是硬编码 context.Background()（不继承调度 HTTP 请求的链路），
// 因此这里在任务入口用全局 Tracer 创建一次任务执行的"根 span"，
// 任务内所有 SQL（repository 已 ctx 贯穿 WithContext）自动挂到该链路下，
// 日志行带 trace_id 可回溯，达到"一次任务执行 = 一条独立 trace"。
// 追踪未启用时返回原函数，零开销、零行为变化。
// 注意：SkyWalking-go（native）通道走编译期 agent 自动埋点，不覆盖 xxl-job（无 HTTP/SQL 之外的
// 自动插件），该通道下 xxl 的 native 手动埋点为 P1 待办。
// 用法：executor.RegTask("job.xxx", traced("job.xxx", XxxJob))
func traced(tr trace.Tracer, handler string, fn xxl.TaskFunc) xxl.TaskFunc {
	return func(cxt context.Context, param *xxl.RunReq) string {
		if tr == nil {
			return fn(cxt, param)
		}
		start := time.Now()
		ctx, span := tr.Start(cxt, "xxl-job."+handler,
			trace.WithAttributes(
				attribute.Int64("xxl.job_id", param.JobID),
				attribute.Int64("xxl.log_id", param.LogID),
				attribute.Int64("xxl.broadcast_index", param.BroadcastIndex),
				attribute.Int64("xxl.broadcast_total", param.BroadcastTotal),
			),
		)
		tid := span.SpanContext().TraceID().String()
		logger.Infof("[xxl:%s] start, trace=%s, jobId=%d, param=%s", handler, tid, param.JobID, truncate(param.ExecutorParams, 512))
		defer func() {
			if r := recover(); r != nil {
				span.RecordError(fmt.Errorf("task panic: %v", r))
				span.SetStatus(codes.Error, "task panic")
				span.End()
				logger.Errorf("[xxl:%s] panic, trace=%s, err=%v", handler, tid, r)
				panic(r) // 交还 xxl 库 recover → 回调调度中心失败
			}
			span.End()
			logger.Infof("[xxl:%s] done, trace=%s, cost=%dms", handler, tid, time.Since(start).Milliseconds())
		}()
		return fn(ctx, param)
	}
}

// truncate 截断超长字符串（日志输出任务参数用，避免刷屏/泄全量）
func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

// TestJob 测试任务 —— 在 XXL-JOB 管理后台新建任务，JobHandler 填 job.test 即可触发
func TestJob(ctx context.Context, param *xxl.RunReq) string {
	logger.Infof("[TestJob] start, param: %s", param.ExecutorParams)
	start := time.Now()

	// TODO: 在此编写具体业务逻辑
	time.Sleep(200 * time.Millisecond)

	logger.Infof("[TestJob] done, cost: %s", time.Since(start))
	return fmt.Sprintf("测试任务执行成功，耗时 %s", time.Since(start))
}

// HealthCheckJob 健康检查任务 —— JobHandler: job.health_check
func HealthCheckJob(ctx context.Context, param *xxl.RunReq) string {
	logger.Infof("[HealthCheckJob] start")
	logger.Infof("[HealthCheckJob] done")
	return "ok"
}
