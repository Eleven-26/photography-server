package middleware

import (
	"github.com/gin-gonic/gin"

	"photography-server/internal/pkg/errs"
	"photography-server/internal/pkg/logger"
	"photography-server/internal/response"
)

// Recovery 统一异常恢复，返回 JSON 错误
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				logger.Errorf("panic recovered: %v", r)
				response.Fail(c, errs.Internal(""))
				c.Abort()
			}
		}()
		c.Next()
	}
}
