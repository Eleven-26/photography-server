package middleware

import (
	v3 "github.com/SkyAPM/go2sky-plugins/gin/v3"
	"github.com/gin-gonic/gin"

	"photography-server/internal/infrastructure"
)

// SkyWalkingTrace SkyWalking 链路追踪中间件。
// tracer 未初始化（enable=false 或初始化失败）时返回 nil，由调用方跳过挂载，
// 保证关闭 SkyWalking 时请求路径零开销、零行为变化。
// 注意：需在路由注册前、其余业务中间件之前挂载，才能覆盖完整请求链路。
func SkyWalkingTrace(engine *gin.Engine) gin.HandlerFunc {
	tracer := infrastructure.SkyWalking()
	if tracer == nil {
		return nil
	}
	return v3.Middleware(engine, tracer)
}
