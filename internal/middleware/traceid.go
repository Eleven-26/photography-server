package middleware

import (
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/trace"
)

// TraceID 将当前请求 entry span 的 trace_id 写入响应头 X-Trace-Id，
// 便于用 trace_id 在 SkyWalking UI / 日志中检索对应链路。
// 必须注册在 SkyWalkingTrace（otelgin）之后才能从请求上下文取到 span；
// 追踪未启用（无有效 span）时不写头，请求路径零行为变化。
func TraceID() gin.HandlerFunc {
	return func(c *gin.Context) {
		if sc := trace.SpanContextFromContext(c.Request.Context()); sc.IsValid() {
			c.Header("X-Trace-Id", sc.TraceID().String())
			c.Set("trace_id", sc.TraceID().String())
		}
		c.Next()
	}
}
