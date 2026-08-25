package middleware

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"

	"photography-server/internal/config"
	"photography-server/internal/model"
	"photography-server/internal/pkg/errs"
	"photography-server/internal/pkg/jwtpkg"
	"photography-server/internal/pkg/logger"
	"photography-server/internal/response"
	"photography-server/internal/service"

	"gorm.io/gorm"
)

type Middlewares struct {
	DB  *gorm.DB
	Cfg *config.Config
}

type ctxKey string

// OperatorKey 操作人上下文键，同时用于 gin.Context 与 request context
const OperatorKey ctxKey = "photography.operator"

// CORS 跨域中间件（前端开发服务器/网关均可访问）
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin == "" {
			origin = "*"
		}
		c.Header("Access-Control-Allow-Origin", origin)
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Requested-With, X-Client")
		c.Header("Access-Control-Max-Age", "86400")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}

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
		if err := m.DB.First(&u, claims.UserID).Error; err != nil {
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

// GetOperator 从 gin 上下文获取当前操作人
func GetOperator(c *gin.Context) service.Operator {
	v, _ := c.Get(string(OperatorKey))
	op, _ := v.(service.Operator)
	return op
}

// RequestLog 请求日志中间件
func RequestLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		dur := time.Since(start).Milliseconds()
		status := c.Writer.Status()
		op := GetOperator(c)
		logger.Infof("%s %s -> %d (%dms) user=%d company=%d",
			c.Request.Method, c.Request.URL.Path, status, dur, op.UserID, op.CompanyID)
	}
}

// OperationLog 操作日志中间件：记录非查询类写操作的审计日志
func (m *Middlewares) OperationLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		dur := time.Since(start).Milliseconds()
		method := c.Request.Method
		if method == "GET" || method == "OPTIONS" || method == "HEAD" {
			return
		}
		op := GetOperator(c)
		if op.UserID == 0 {
			return
		}
		log := model.SysOperationLog{
			TenantBase: model.TenantBase{
				Base:      model.Base{CreatedBy: op.UserID, UpdatedBy: op.UserID},
				CompanyID: op.CompanyID,
			},
			UserID:   op.UserID,
			Username: op.Username,
			Module:   c.GetHeader("X-Module"),
			Action:   c.GetHeader("X-Action"),
			Method:   method,
			Path:     c.Request.URL.Path,
			Params:   c.Request.URL.RawQuery,
			IP:       c.ClientIP(),
			Status:   1,
			Duration: dur,
		}
		if c.Writer.Status() >= 400 {
			log.Status = 0
		}
		// 异步写入，失败不影响主流程
		m.DB.Create(&log)
	}
}
