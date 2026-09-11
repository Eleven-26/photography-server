package errs

import (
	"fmt"
	"net/http"
	"runtime"
	"strings"
)

// BizError 业务错误。对外契约只有 Code 与 Msg（Msg 必须是对外安全文案）。
// 5xx 系错误额外携带 cause（原始底层错误）与 stack（构造点调用栈）：
// 两者仅进服务端日志与 Jaeger 链路（exception.stacktrace），不回显给客户端，
// 使「控制台有原文、Jaeger 有行号」而响应体保持通用文案。
type BizError struct {
	Code  int
	Msg   string
	cause error     // 原始底层错误（仅 5xx 系）
	stack []uintptr // 构造点调用栈（仅 5xx 系抓取；4xx 为 nil，零开销）
}

func (e *BizError) Error() string {
	if e.cause != nil {
		return e.Msg + ": " + e.cause.Error()
	}
	return e.Msg
}

// Unwrap 标准错误链：errors.Is/As 可穿透到 cause
func (e *BizError) Unwrap() error { return e.cause }

// StackTrace 构造点堆栈的格式化文本（file:line + 函数名），写入 Jaeger exception.stacktrace
func (e *BizError) StackTrace() string {
	if len(e.stack) == 0 {
		return ""
	}
	var b strings.Builder
	frames := runtime.CallersFrames(e.stack)
	for {
		f, more := frames.Next()
		fmt.Fprintf(&b, "%s\n\t%s:%d\n", f.Function, f.File, f.Line)
		if !more {
			break
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

// callerPCs 抓取调用方（跳过自身与构造函数共 skip 层）的程序计数器
func callerPCs(skip int) []uintptr {
	pcs := make([]uintptr, 24)
	n := runtime.Callers(skip+1, pcs)
	return pcs[:n]
}

func New(code int, msg string) *BizError {
	return &BizError{Code: code, Msg: msg}
}

// Common business errors. HTTP status defaults to 200 (business code distinguishes),
// the httpStatus is only used when the error must also affect the HTTP layer.
func BadRequest(msg string) *BizError {
	return New(40000, msg)
}

func Unauthorized(msg string) *BizError {
	if msg == "" {
		msg = "登录已过期，请重新登录"
	}
	return &BizError{Code: 40100, Msg: msg}
}

func Forbidden(msg string) *BizError {
	if msg == "" {
		msg = "无权限操作"
	}
	return &BizError{Code: 40300, Msg: msg}
}

func NotFound(msg string) *BizError {
	if msg == "" {
		msg = "资源不存在"
	}
	return &BizError{Code: 40400, Msg: msg}
}

func Conflict(msg string) *BizError {
	return New(40900, msg)
}

func Internal(msg string) *BizError {
	if msg == "" {
		msg = "系统繁忙，请稍后再试"
	}
	return &BizError{Code: 50000, Msg: msg, stack: callerPCs(1)}
}

// InternalWrap 包装底层错误：对外仍是通用文案，cause 保留进日志与 Jaeger 链路。
// service/repository 层捕获底层 err 后应优先用它替换 Internal("")，避免吞掉原文。
func InternalWrap(cause error) *BizError {
	if cause == nil {
		return Internal("")
	}
	return &BizError{Code: 50000, Msg: "系统繁忙，请稍后再试", cause: cause, stack: callerPCs(1)}
}

// InternalWrapMsg 同 InternalWrap，但自定义对外安全文案（如「短信服务暂不可用」）
func InternalWrapMsg(cause error, msg string) *BizError {
	if cause == nil {
		return Internal(msg)
	}
	if msg == "" {
		msg = "系统繁忙，请稍后再试"
	}
	return &BizError{Code: 50000, Msg: msg, cause: cause, stack: callerPCs(1)}
}

func HTTPStatus(e error) int {
	if be, ok := e.(*BizError); ok {
		switch {
		case be.Code >= 40100 && be.Code < 40200:
			return http.StatusUnauthorized
		case be.Code >= 40300 && be.Code < 40400:
			return http.StatusForbidden
		case be.Code >= 40400 && be.Code < 40500:
			return http.StatusNotFound
		case be.Code >= 40900 && be.Code < 41000:
			return http.StatusConflict
		case be.Code >= 40000 && be.Code < 40100:
			return http.StatusBadRequest
		case be.Code >= 50000:
			return http.StatusInternalServerError
		}
		return http.StatusOK
	}
	return http.StatusInternalServerError
}
