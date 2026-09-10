// Package params 统一请求参数取值。
//
// 项目 API 除 /health 外全部为 POST JSON（见 internal/router/router.go），
// 规范要求：**POST 的提交参数一律从 JSON body 取，不再读 query**。
// 此前列表接口用 c.Query 读取分页与筛选参数，前端若把参数放在 body 会被静默忽略，
// 导致分页、搜索、状态筛选整体失效（契约断裂）。本包把取值语义统一到 body，
// 从根上消除这类问题；GET/HEAD 等无 body 的方法仍走 query，保持通用。
//
// 用法：
//
//	page := params.Int(c, "page")
//	keyword := params.Str(c, "keyword")
//
// 需在路由上注册 params.Middleware()（见 router.New）。
package params

import (
	"bytes"
	"encoding/json"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// ctxKeyBody 缓存到 gin.Context 的请求体参数表键名
const ctxKeyBody = "params.body"

// Middleware 预解析 JSON 请求体为参数表并缓存，随后回填 body 供业务侧 ShouldBindJSON 正常使用。
//
// 仅处理带 JSON Content-Type（或未声明 Content-Type）的 POST/PUT/PATCH；
// multipart 等非 JSON 请求原样放行，不消费流，避免破坏文件上传。
func Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		switch c.Request.Method {
		case http.MethodPost, http.MethodPut, http.MethodPatch:
			if ct := c.GetHeader("Content-Type"); ct == "" || strings.Contains(ct, "json") {
				cache(c)
			}
		}
		c.Next()
	}
}

// cache 读取 body → 回填 → 解析为参数表存入 context。
// 解析失败（非 JSON、数组、空体）时静默跳过，仅丢失参数表，不影响请求继续执行。
func cache(c *gin.Context) {
	if c.Request.Body == nil {
		return
	}
	raw, err := io.ReadAll(c.Request.Body)
	_ = c.Request.Body.Close()
	if err != nil {
		return
	}
	// 无论解析成败都回填：业务侧 ShouldBindJSON / c.GetRawData 必须仍能读到完整 body
	c.Request.Body = io.NopCloser(bytes.NewReader(raw))
	if len(raw) == 0 {
		return
	}
	var m map[string]interface{}
	if err := json.Unmarshal(raw, &m); err != nil || m == nil {
		return
	}
	c.Set(ctxKeyBody, m)
}

// rawValue 按方法语义取原始值：写方法（POST/PUT/PATCH）只认 body，其余方法读 query。
// 写方法下即便 body 中不存在该键也**不**回退 query，避免"参数写错位置也悄悄生效"的假象。
func rawValue(c *gin.Context, key string) (interface{}, bool) {
	switch c.Request.Method {
	case http.MethodPost, http.MethodPut, http.MethodPatch:
		if v, ok := c.Get(ctxKeyBody); ok {
			if m, ok := v.(map[string]interface{}); ok {
				val, exist := m[key]
				return val, exist
			}
		}
		return nil, false
	default:
		return c.GetQuery(key)
	}
}

// Str 取字符串参数。body 中的数字/布尔会按字面量转换（如 1 → "1"、true → "true"）。
func Str(c *gin.Context, key string) string {
	v, ok := rawValue(c, key)
	if !ok || v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	case bool:
		return strconv.FormatBool(t)
	case float64:
		return formatNumber(t)
	case json.Number:
		return t.String()
	default:
		b, err := json.Marshal(t)
		if err != nil {
			return ""
		}
		return string(b)
	}
}

// Int 取整型参数，缺失或无法解析时返回 0。
func Int(c *gin.Context, key string) int {
	v, _ := rawValue(c, key)
	return int(toInt64(v))
}

// Int64 取 64 位整型参数，缺失或无法解析时返回 0。
func Int64(c *gin.Context, key string) int64 {
	v, _ := rawValue(c, key)
	return toInt64(v)
}

// formatNumber 把 JSON 数字还原成字面量字符串（整数不带小数点，避免 "10" 变成 "10.000000"）。
func formatNumber(f float64) string {
	if !math.IsInf(f, 0) && !math.IsNaN(f) && f == math.Trunc(f) && math.Abs(f) < 1e15 {
		return strconv.FormatInt(int64(f), 10)
	}
	return strconv.FormatFloat(f, 'f', -1, 64)
}

// toInt64 兼容 body 数字（float64）、字符串数字、布尔与 json.Number
func toInt64(v interface{}) int64 {
	switch t := v.(type) {
	case float64:
		return int64(t)
	case string:
		s := strings.TrimSpace(t)
		if n, err := strconv.ParseInt(s, 10, 64); err == nil {
			return n
		}
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return int64(f)
		}
		return 0
	case bool:
		if t {
			return 1
		}
		return 0
	case json.Number:
		n, _ := t.Int64()
		return n
	default:
		return 0
	}
}
