package middleware

import (
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"photography-server/internal/infrastructure"
	"photography-server/internal/model"
	"photography-server/internal/pkg/logger"
)

// RequestLog 请求日志中间件
// 注意：须注册在 SkyWalkingTrace 之前也能取到 trace_id ——
// 日志在 c.Next() 之后输出，此时链路中间件已执行完，span 已注入请求上下文。
func RequestLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		dur := time.Since(start).Milliseconds()
		status := c.Writer.Status()
		op := GetOperator(c)
		tid := CurrentTraceID(c)
		if tid == "" {
			tid = "-"
		}
		logger.Infof("%s %s -> %d (%dms) trace=%s user=%d company=%d",
			c.Request.Method, c.Request.URL.Path, status, dur, tid, op.UserID, op.CompanyID)
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
			Params:   sanitizeQuery(c.Request.URL),
			IP:       c.ClientIP(),
			Status:   1,
			Duration: dur,
		}
		if c.Writer.Status() >= 400 {
			log.Status = 0
		}
		// 同步写入：透传请求 ctx（WithContext）使该条 SQL 的 span 挂在当前 HTTP 请求链路下，
		// 而不是以独立 trace 入库 —— 与 repository 层 ctx 贯穿的约定保持一致。
		// gin 请求处理链未结束时 Request.Context() 仍有效，可直接复用。
		if err := infrastructure.MySQL().WithContext(c.Request.Context()).Create(&log).Error; err != nil {
			logger.Warnf("operation log write failed: %v", err)
		}
	}
}

// sanitizeQuery 序列化请求查询串并剔除敏感参数。
// 认证支持 ?token=（auth.go），若原样记录 URL.RawQuery 会把 JWT 写入审计库，
// 这里统一做脱敏：任何含 token/secret/password/sign 的查询参数一律不落库。
func sanitizeQuery(u *url.URL) string {
	q := u.Query()
	if len(q) == 0 {
		return ""
	}
	for k := range q {
		lower := strings.ToLower(k)
		if strings.Contains(lower, "token") || strings.Contains(lower, "secret") ||
			strings.Contains(lower, "password") || strings.Contains(lower, "sign") {
			q.Del(k)
		}
	}
	return q.Encode()
}
