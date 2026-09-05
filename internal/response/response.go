package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
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
func Fail(c *gin.Context, err error) {
	be, ok := err.(*errs.BizError)
	if !ok {
		logger.Errorf("[response] internal error: path=%s err=%v", c.Request.URL.Path, err)
		be = errs.Internal("") // 默认文案：系统繁忙，请稍后再试
	}
	c.JSON(errs.HTTPStatus(err), Body{Code: be.Code, Msg: be.Msg, Data: nil, TraceID: traceIDOf(c)})
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
