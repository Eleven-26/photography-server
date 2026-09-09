package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// CORS 跨域中间件（前端开发服务器/网关均可访问）。
// 安全策略：只回显白名单内的 Origin（origins 来自 app.cors_origins 配置），
// 且仅当 Origin 命中白名单时才允许携带凭据 —— 杜绝"任意 Origin 回显 + Allow-Credentials"
// 导致的跨站携带凭据访问。非浏览器客户端（无 Origin 头/同源/服务端调用）不受影响。
func CORS(origins []string) gin.HandlerFunc {
	allow := make(map[string]struct{}, len(origins))
	for _, o := range origins {
		if o = strings.TrimSpace(o); o != "" {
			allow[o] = struct{}{}
		}
	}
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" {
			if _, ok := allow[origin]; ok {
				c.Header("Access-Control-Allow-Origin", origin)
				c.Header("Access-Control-Allow-Credentials", "true")
				c.Header("Vary", "Origin")
			}
			// 未命中白名单：不加任何 ACAO 头，浏览器会拦截跨域响应
		}
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
