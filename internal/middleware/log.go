package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"photography-server/internal/infrastructure"
	"photography-server/internal/model"
	"photography-server/internal/pkg/logger"
)

// RequestLog 请求日志中间件
// 注意：须注册在 JaegerTrace / agent 入口之前也能取到 trace_id ——
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
// 修复（#33）：记录 POST body（JSON 格式），脱敏后存入 Params 字段
func (m *Middlewares) OperationLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// 读取并缓存请求 body（仅 JSON，限制 10KB 避免记录文件上传）
		var bodyStr string
		if c.Request.Method == "POST" && c.ContentType() == "application/json" {
			if c.Request.ContentLength > 0 && c.Request.ContentLength <= 10*1024 {
				bodyBytes, err := io.ReadAll(c.Request.Body)
				if err == nil {
					// 脱敏后记录
					bodyStr = sanitizeJSON(bodyBytes)
					// 重新设置 body，供后续处理器读取
					c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
				}
			}
		}

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

		// 合并 query string 和 body
		params := sanitizeQuery(c.Request.URL)
		if bodyStr != "" {
			if params != "" {
				params += "&"
			}
			params += "body=" + bodyStr
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
			Params:   params,
			IP:       c.ClientIP(),
			Status:   1,
			Duration: dur,
		}
		if c.Writer.Status() >= 400 {
			log.Status = 0
		}
		// 同步写入：透传请求 ctx（WithContext）使该条 SQL 的 span 挂在当前 HTTP 请求链路下，
		// 而不是以独立 trace 入库 —— 与 repository 层 ctx 贯穿的约定保持一致
		// （Jaeger 通道由 gorm OTel 插件产生 SQL span，native 通道由注入 agent 增强 database/sql，
		//   两者都依赖 ctx 里带有父 span，此约定为两条通道共用的前提）。
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

// sanitizeJSON 对 JSON body 做脱敏：删除含 password/secret/token/sign 的字段
func sanitizeJSON(data []byte) string {
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		// 非 JSON 或解析失败，返回空串
		return ""
	}
	for k := range m {
		lower := strings.ToLower(k)
		if strings.Contains(lower, "password") || strings.Contains(lower, "secret") ||
			strings.Contains(lower, "token") || strings.Contains(lower, "sign") ||
			strings.Contains(lower, "credential") {
			delete(m, k)
		}
	}
	// 重新序列化为紧凑格式
	out, err := json.Marshal(m)
	if err != nil {
		return ""
	}
	return string(out)
}
