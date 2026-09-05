package middleware

import (
	"github.com/gin-gonic/gin"
	otelgin "go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"

	"photography-server/internal/infrastructure"
)

// SkyWalkingTrace SkyWalking 链路追踪中间件（otelgin，为每个 HTTP 请求产生 entry span）。
// SkyWalking 未启用（enable=false 或初始化失败）时返回 nil，由调用方跳过挂载，
// 保证关闭追踪时请求路径零开销、零行为变化。
// 注意：需在路由注册前、其余业务中间件之前挂载，才能覆盖完整请求链路。
func SkyWalkingTrace(service string) gin.HandlerFunc {
	if !infrastructure.SkyWalkingEnabled() {
		return nil
	}
	return otelgin.Middleware(service)
}
