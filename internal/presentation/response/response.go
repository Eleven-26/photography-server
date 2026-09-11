package response

import (
	"errors"
	"mime"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"photography-server/internal/pkg/errs"
	"photography-server/internal/pkg/logger"
)

type Body struct {
	Code    int         `json:"code"`
	Msg     string      `json:"msg"`
	Data    interface{} `json:"data"`
	TraceID string      `json:"trace_id"` // 当前请求链路 trace_id，便于前端按链路反馈问题
}

type Page struct {
	List     interface{} `json:"list"`
	Total    int64       `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"page_size"`
}

func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Body{Code: 0, Msg: "ok", Data: data, TraceID: traceIDOf(c)})
}

func OKNil(c *gin.Context) {
	OK(c, nil)
}

func PageOK(c *gin.Context, list interface{}, total int64, page, pageSize int) {
	OK(c, Page{List: list, Total: total, Page: page, PageSize: pageSize})
}

// Fail 统一错误响应。业务错误（*errs.BizError）按定义透出；
// 非业务错误（内部异常）不回显内部细节，仅返回通用文案，完整错误记入服务端日志。
// 错误响应同样带 trace_id，前端可用该 ID 反馈问题到后端精确检索链路。
// 5xx 一律落 Errorf 日志（含构造点堆栈）并写入 Jaeger span（exception.stacktrace）——
// 原先 BizError 分支静默导致吞错误无从排查（2026-09-11 dashboard/overview 复盘）。
func Fail(c *gin.Context, err error) {
	var be *errs.BizError
	if !errors.As(err, &be) {
		be = errs.InternalWrap(err) // 带构造点堆栈；Error() 保留原始错误文本
	}
	if be.Code >= 50000 {
		logger.Errorf("[response] server error: path=%s trace=%s err=%v\nstack: %s",
			c.Request.URL.Path, traceIDOf(c), err, be.StackTrace())
		recordServerSpan(c, err, be)
	}
	c.JSON(errs.HTTPStatus(err), Body{Code: be.Code, Msg: be.Msg, Data: nil, TraceID: traceIDOf(c)})
}

// recordServerSpan 把 5xx 错误写入当前请求 span：状态置 Error + exception 事件
// （message=错误原文，stacktrace=errs 构造点行号）。链路未启用或未采样时为 no-op。
func recordServerSpan(c *gin.Context, rawErr error, be *errs.BizError) {
	span := trace.SpanFromContext(c.Request.Context())
	if !span.IsRecording() {
		return
	}
	span.SetStatus(codes.Error, be.Msg)
	attrs := []attribute.KeyValue{}
	if st := be.StackTrace(); st != "" {
		attrs = append(attrs, attribute.String("exception.stacktrace", st))
	}
	span.RecordError(rawErr, trace.WithAttributes(attrs...))
}

// File 二进制/文本文件下载。filename 含中文时按 RFC 5987 生成 filename*，
// 前端从 Content-Disposition 取文件名即可（见前端 download() 助手）。
func File(c *gin.Context, filename, contentType string, content []byte) {
	c.Header("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": filename}))
	c.Header("Content-Length", strconv.Itoa(len(content)))
	c.Data(http.StatusOK, contentType, content)
}

// traceIDOf 取当前请求的 trace_id：优先请求上下文里的 entry span，
// 回退 TraceID 中间件放入 gin context 的值；链路未启用时返回空串。
// 注意：不能依赖 middleware 包（middleware -> response 会形成循环依赖），此处独立实现。
func traceIDOf(c *gin.Context) string {
	if sc := trace.SpanContextFromContext(c.Request.Context()); sc.IsValid() {
		return sc.TraceID().String()
	}
	return c.GetString("trace_id")
}
