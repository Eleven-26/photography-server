package middleware

import (
	"context"

	"github.com/gin-gonic/gin"

	"photography-server/internal/infrastructure"
	"photography-server/internal/model"
	"photography-server/internal/pkg/errs"
	"photography-server/internal/pkg/jwtpkg"
	"photography-server/internal/response"
	"photography-server/internal/service"
)

// extractToken 从 Authorization: Bearer 或 token 参数中提取令牌
func extractToken(c *gin.Context) string {
	auth := c.GetHeader("Authorization")
	if len(auth) > 7 && auth[:7] == "Bearer " {
		return auth[7:]
	}
	return c.Query("token")
}

// Auth JWT 认证中间件，校验令牌并注入操作人上下文
func (m *Middlewares) Auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractToken(c)
		if token == "" {
			response.Fail(c, errs.Unauthorized(""))
			c.Abort()
			return
		}
		claims, err := jwtpkg.Parse(m.Cfg.JWT.Secret, m.Cfg.JWT.Issuer, token)
		if err != nil {
			response.Fail(c, errs.Unauthorized(""))
			c.Abort()
			return
		}
		var u model.SysUser
		if err := infrastructure.MySQL().First(&u, claims.UserID).Error; err != nil {
			response.Fail(c, errs.Unauthorized(""))
			c.Abort()
			return
		}
		if u.Status != 1 {
			response.Fail(c, errs.Forbidden("账号已被停用"))
			c.Abort()
			return
		}
		op := service.Operator{
			UserID:    u.ID,
			Username:  u.Username,
			Nickname:  u.Nickname,
			CompanyID: u.CompanyID,
			StoreID:   u.StoreID,
			RoleID:    u.RoleID,
		}
		c.Set(string(OperatorKey), op)
		ctx := context.WithValue(c.Request.Context(), OperatorKey, op)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
