package params

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// newCtx 构造测试上下文：method / Content-Type / body / query
func newCtx(t *testing.T, method, contentType, body, query string) *gin.Context {
	t.Helper()
	gin.SetMode(gin.TestMode)
	url := "/x"
	if query != "" {
		url += "?" + query
	}
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(method, url, bytes.NewBufferString(body))
	if contentType != "" {
		c.Request.Header.Set("Content-Type", contentType)
	}
	return c
}

// POST 参数只认 body：同名 query 不得覆盖，body 缺失的键不回退 query
func TestPostTakesBodyOnly(t *testing.T) {
	c := newCtx(t, http.MethodPost, "application/json",
		`{"page":2,"page_size":50,"keyword":"张三","unread":1,"flag":true}`,
		"page=9&keyword=fromQuery")
	Middleware()(c)

	if got := Int(c, "page"); got != 2 {
		t.Fatalf("page = %d, want 2（body 优先，不能被 query 覆盖）", got)
	}
	if got := Int(c, "page_size"); got != 50 {
		t.Fatalf("page_size = %d, want 50", got)
	}
	if got := Str(c, "keyword"); got != "张三" {
		t.Fatalf("keyword = %q, want 张三", got)
	}
	// 数字/布尔按字面量转换
	if got := Str(c, "unread"); got != "1" {
		t.Fatalf("unread = %q, want \"1\"", got)
	}
	if got := Str(c, "flag"); got != "true" {
		t.Fatalf("flag = %q, want \"true\"", got)
	}
	// body 中不存在的键**不**回退 query
	if got := Str(c, "keyword_missing"); got != "" {
		t.Fatalf("缺失键 = %q, want 空（POST 不读 query）", got)
	}
}

// 中间件读走 body 后必须回填，业务侧 ShouldBindJSON 仍需拿到完整字段
func TestBodyRefilledForBindJSON(t *testing.T) {
	type req struct {
		Page      int    `json:"page"`
		PageSize  int    `json:"page_size"`
		Keyword   string `json:"keyword"`
		Status    int    `json:"status"`
		StartDate string `json:"start_date"`
	}
	c := newCtx(t, http.MethodPost, "application/json",
		`{"page":3,"page_size":15,"keyword":"abc","status":2,"start_date":"2026-09-01"}`, "")
	Middleware()(c)

	var r req
	if err := c.ShouldBindJSON(&r); err != nil {
		t.Fatalf("ShouldBindJSON 失败（body 未正确回填）: %v", err)
	}
	if r.Page != 3 || r.PageSize != 15 || r.Keyword != "abc" || r.Status != 2 || r.StartDate != "2026-09-01" {
		t.Fatalf("绑定结果不完整: %+v", r)
	}
	// 同一请求内 params 与 ShouldBindJSON 可共存
	if got := Int(c, "page"); got != 3 {
		t.Fatalf("params.Int(page) = %d, want 3", got)
	}
}

// GET 等无 body 方法保持 query 语义
func TestGetUsesQuery(t *testing.T) {
	c := newCtx(t, http.MethodGet, "", "", "page=7&keyword=k&photographer_id=12")
	Middleware()(c)

	if got := Int(c, "page"); got != 7 {
		t.Fatalf("page = %d, want 7", got)
	}
	if got := Str(c, "keyword"); got != "k" {
		t.Fatalf("keyword = %q, want k", got)
	}
	if got := Int64(c, "photographer_id"); got != 12 {
		t.Fatalf("photographer_id = %d, want 12", got)
	}
}

// 非 JSON（如 multipart 上传）不解析、不消费流
func TestNonJSONSkipped(t *testing.T) {
	const payload = "--b\r\nContent-Disposition: form-data; name=\"page\"\r\n\r\n5\r\n--b--\r\n"
	c := newCtx(t, http.MethodPost, "multipart/form-data; boundary=b", payload, "")
	Middleware()(c)

	if got := Str(c, "page"); got != "" {
		t.Fatalf("multipart 不应被解析为参数表, got %q", got)
	}
	// body 未被消费，业务侧仍可读取
	buf := make([]byte, len(payload))
	if _, err := c.Request.Body.Read(buf); err != nil {
		t.Fatalf("body 被提前消费: %v", err)
	}
	if string(buf) != payload {
		t.Fatalf("body 内容被破坏")
	}
}

// 非法/空 body 不应 panic，仅无参数可用
func TestInvalidBodyNoPanic(t *testing.T) {
	for _, body := range []string{"", "not-json", "[1,2,3]", "null"} {
		c := newCtx(t, http.MethodPost, "application/json", body, "")
		Middleware()(c)
		if got := Str(c, "page"); got != "" {
			t.Fatalf("body=%q 时不应取到参数, got %q", body, got)
		}
	}
}

// 类型容错：字符串数字、空串、布尔
func TestTypeTolerance(t *testing.T) {
	c := newCtx(t, http.MethodPost, "application/json",
		`{"a":"12","b":"","c":true,"d":0,"e":-3,"f":3.9}`, "")
	Middleware()(c)

	if got := Int(c, "a"); got != 12 {
		t.Fatalf("字符串数字 a = %d, want 12", got)
	}
	if got := Int(c, "b"); got != 0 {
		t.Fatalf("空串 b = %d, want 0", got)
	}
	if got := Int(c, "c"); got != 1 {
		t.Fatalf("布尔 c = %d, want 1", got)
	}
	if got := Int64(c, "e"); got != -3 {
		t.Fatalf("负数 e = %d, want -3", got)
	}
	if got := Int(c, "f"); got != 3 {
		t.Fatalf("小数 f = %d, want 3", got)
	}
}

// 引擎级：真实 gin 路由链路下，params 取参与 ShouldBindJSON 必须同时生效，
// 且 URL 上的同名 query 不得干扰（这是列表接口分页/筛选的核心契约）。
func TestEngineLevelPostBodyWins(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(Middleware())
	engine.POST("/order/list", func(c *gin.Context) {
		var req struct {
			Page    int    `json:"page"`
			Status  int    `json:"status"`
			Keyword string `json:"keyword"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.String(http.StatusBadRequest, "bind error: %v", err)
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"page":      Int(c, "page"),
			"page_size": Int(c, "page_size"),
			"status":    Str(c, "status"),
			"keyword":   Str(c, "keyword"),
			"boundPage": req.Page,
			"boundSt":   req.Status,
			"boundKw":   req.Keyword,
		})
	})

	body := `{"page":2,"page_size":30,"status":1,"keyword":"李四"}`
	req := httptest.NewRequest(http.MethodPost, "/order/list?page=99&keyword=queryNoise", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("状态码 = %d, body = %s", w.Code, w.Body.String())
	}
	var got map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("响应解析失败: %v", err)
	}
	if got["page"].(float64) != 2 || got["page_size"].(float64) != 30 {
		t.Fatalf("分页取值错误: %v", got)
	}
	if got["keyword"] != "李四" {
		t.Fatalf("keyword = %v, want 李四（query 干扰项必须被忽略）", got["keyword"])
	}
	if got["status"] != "1" {
		t.Fatalf("status = %v, want \"1\"", got["status"])
	}
	if got["boundPage"].(float64) != 2 || got["boundKw"] != "李四" || got["boundSt"].(float64) != 1 {
		t.Fatalf("ShouldBindJSON 结果不完整: %v", got)
	}
}
