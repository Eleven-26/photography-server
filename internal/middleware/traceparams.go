package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const (
	traceBodyMaxBytes = 4 << 10 // 单请求最多记录 4KB body，超出截断
)

const traceBodyTruncMark = "...(truncated)"

// sensitiveKeySubstrings 命中即脱敏的 key 子串（小写包含匹配）
var sensitiveKeySubstrings = []string{"password", "pwd", "secret", "token", "authorization", "credential"}

// TraceParams 将请求参数记录到 entry span 的属性上：
//   - query string -> http.query
//   - JSON body    -> http.request.body（限 4KB，敏感字段值脱敏为 ****）
//
// 必须注册在 SkyWalkingTrace（otelgin）之后、业务处理器之前：
// 前者保证能从请求上下文取到 entry span，后者保证 body 先读取再复原、业务不受影响。
func TraceParams() gin.HandlerFunc {
	return func(c *gin.Context) {
		span := trace.SpanFromContext(c.Request.Context())
		if span.IsRecording() {
			if q := c.Request.URL.RawQuery; q != "" {
				span.SetAttributes(attribute.String("http.query", q))
			}
			if body := captureBody(c); body != "" {
				span.SetAttributes(attribute.String("http.request.body", maskSensitive(body)))
			}
		}
		c.Next()
	}
}

// captureBody 读取 JSON 请求 body 并立即复原（后续处理器不受影响），超限截断
func captureBody(c *gin.Context) string {
	if c.Request.Body == nil {
		return ""
	}
	ct := c.GetHeader("Content-Type")
	if !strings.Contains(ct, "application/json") && !strings.Contains(ct, "text/json") {
		return ""
	}
	data, err := io.ReadAll(io.LimitReader(c.Request.Body, traceBodyMaxBytes+1))
	_ = c.Request.Body.Close()
	if err != nil {
		return ""
	}
	c.Request.Body = io.NopCloser(bytes.NewReader(data))
	if len(data) > traceBodyMaxBytes {
		return string(data[:traceBodyMaxBytes]) + traceBodyTruncMark
	}
	return string(data)
}

// maskSensitive 对 JSON body 中的敏感字段值打码；非 JSON 或解析失败时原样返回（已截断）
func maskSensitive(body string) string {
	var v interface{}
	if err := json.Unmarshal([]byte(body), &v); err != nil {
		return body
	}
	out, err := json.Marshal(maskValue(v))
	if err != nil {
		return body
	}
	return string(out)
}

func maskValue(v interface{}) interface{} {
	switch t := v.(type) {
	case map[string]interface{}:
		for k, val := range t {
			if isSensitiveKey(k) {
				t[k] = "****"
				continue
			}
			t[k] = maskValue(val)
		}
		return t
	case []interface{}:
		for i, item := range t {
			t[i] = maskValue(item)
		}
		return t
	default:
		return v
	}
}

func isSensitiveKey(key string) bool {
	k := strings.ToLower(key)
	for _, s := range sensitiveKeySubstrings {
		if strings.Contains(k, s) {
			return true
		}
	}
	return false
}
