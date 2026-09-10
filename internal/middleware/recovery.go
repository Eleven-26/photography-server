package middleware

import (
	"runtime/debug"

	"github.com/gin-gonic/gin"

	"photography-server/internal/pkg/errs"
	"photography-server/internal/pkg/logger"
	"photography-server/internal/presentation/response"
)

// Recovery 统一异常恢复，返回 JSON 错误
// 修复（#36）：记录完整堆栈信息，便于生产环境定位 panic 位置
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				stack := string(debug.Stack())
				logger.Errorf("panic recovered: %v\nstack: %s\nreq: %s %s", r, stack, c.Request.Method, c.Request.URL.Path)
				response.Fail(c, errs.Internal(""))
				c.Abort()
			}
		}()
		c.Next()
	}
}
