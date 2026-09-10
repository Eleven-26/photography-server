package middleware

import (
	"github.com/gin-gonic/gin"
	otelgin "go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

// JaegerTrace Jaeger（OTel）通道的 HTTP 入口中间件（otelgin，为每个 HTTP 请求产生 entry span）。
// 链路未启用（m.Tracer == nil，即 jaeger.enable=false 或初始化失败）时返回 nil，由调用方跳过挂载，
// 保证关闭追踪时请求路径零开销、零行为变化。
// 注意：需在路由注册前、其余业务中间件之前挂载，才能覆盖完整请求链路。
// SkyWalking-go（native）通道不走本中间件：其 gin 入口 span 由编译期注入的 agent 自动创建。
func (m *Middlewares) JaegerTrace() gin.HandlerFunc {
	if m.Tracer == nil {
		return nil
	}
	return otelgin.Middleware(m.Cfg.Jaeger.Service)
}
