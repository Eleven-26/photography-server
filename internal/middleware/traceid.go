package middleware

import (
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/trace"

	skywtrace "github.com/apache/skywalking-go/toolkit/trace"
)

// CurrentTraceID 返回当前请求的 trace_id（无有效 span 时为空串）。
// 取值优先级：
//  1. OpenTelemetry 通道（Jaeger 版）：请求上下文里的 entry span；
//  2. gin context 缓存值（TraceID 中间件已写入，SkyWalking-go 版为 agent native trace id）；
//  3. skywalking-go agent 当前 goroutine 上下文（toolkit，仅注入构建且有活跃 span 时非空；
//     普通构建下 toolkit 为安全空实现，返回空串，不影响请求路径）。
func CurrentTraceID(c *gin.Context) string {
	if sc := trace.SpanContextFromContext(c.Request.Context()); sc.IsValid() {
		return sc.TraceID().String()
	}
	if tid := c.GetString("trace_id"); tid != "" {
		return tid
	}
	return skywtrace.GetTraceID()
}

// TraceID 将当前请求 entry span 的 trace_id 写入响应头 X-Trace-Id，
// 便于用 trace_id 在 Jaeger UI / SkyWalking(Horizon) UI / 日志中检索对应链路。
// 必须注册在 JaegerTrace（otelgin）之后才能从请求上下文取到 span；
// 追踪未启用（无有效 span）时不写头，请求路径零行为变化。
func TraceID() gin.HandlerFunc {
	return func(c *gin.Context) {
		if tid := CurrentTraceID(c); tid != "" {
			c.Header("X-Trace-Id", tid)
			c.Set("trace_id", tid)
		}
		c.Next()
	}
}
