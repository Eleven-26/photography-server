package errs

import (
	"net/http"
)

type BizError struct {
	Code int
	Msg  string
}

func (e *BizError) Error() string {
	return e.Msg
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
	return &BizError{Code: 50000, Msg: msg}
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
		case be.Code >= 40000 && be.Code < 40100:
			return http.StatusBadRequest
		case be.Code >= 50000:
			return http.StatusInternalServerError
		}
		return http.StatusOK
	}
	return http.StatusInternalServerError
}
