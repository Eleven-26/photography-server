package middleware

import (
	"time"

	"github.com/gin-gonic/gin"

	"photography-server/internal/infrastructure"
	"photography-server/internal/model"
	"photography-server/internal/pkg/logger"
)

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
		infrastructure.MySQL().Create(&log)
	}
}
