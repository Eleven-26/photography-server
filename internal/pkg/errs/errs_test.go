package errs

import (
	"errors"
	"strings"
	"testing"
)

// 5xx 系错误必须带构造点堆栈与 cause 透传——这是「控制台有原文、Jaeger 有行号」的根基
func TestInternalStackAndCause(t *testing.T) {
	be := Internal("")
	if be.Code != 50000 || be.Msg != "系统繁忙，请稍后再试" {
		t.Fatalf("Internal 基础字段错误: code=%d msg=%s", be.Code, be.Msg)
	}
	st := be.StackTrace()
	if st == "" {
		t.Fatal("Internal 未抓取构造点堆栈")
	}
	if !strings.Contains(st, "errs_test.go") || !strings.Contains(st, "errs.go") {
		t.Fatalf("堆栈未包含构造点文件行号:\n%s", st)
	}

	cause := errors.New("sql: no rows")
	wrap := InternalWrap(cause)
	if !errors.Is(wrap, cause) {
		t.Fatal("InternalWrap 未通过 Unwrap 透传 cause")
	}
	if wrap.Error() != "系统繁忙，请稍后再试: sql: no rows" {
		t.Fatalf("Error() 未拼接 cause: %q", wrap.Error())
	}
	if wrap.Msg != "系统繁忙，请稍后再试" {
		t.Fatalf("对外 Msg 必须保持安全文案: %q", wrap.Msg)
	}
	if wrap.StackTrace() == "" {
		t.Fatal("InternalWrap 未抓取构造点堆栈")
	}

	msg := InternalWrapMsg(cause, "短信服务暂不可用，请稍后再试")
	if msg.Msg != "短信服务暂不可用，请稍后再试" || !errors.Is(msg, cause) {
		t.Fatalf("InternalWrapMsg 自定义文案失效: %q", msg.Msg)
	}
	if msg.Error() != "短信服务暂不可用，请稍后再试: sql: no rows" {
		t.Fatalf("InternalWrapMsg Error() 未拼接 cause: %q", msg.Error())
	}
}

// 4xx 是正常业务流：不抓栈（零开销）、Error() 不带 cause
func TestBizErrorNoStack(t *testing.T) {
	be := BadRequest("参数错误")
	if be.StackTrace() != "" {
		t.Fatalf("4xx 不应抓取堆栈: %s", be.StackTrace())
	}
	if be.Error() != "参数错误" {
		t.Fatalf("4xx Error() 应仅返回 Msg: %q", be.Error())
	}
	if HTTPStatus(be) != 400 {
		t.Fatalf("40000 应映射 HTTP 400: %d", HTTPStatus(be))
	}
	if HTTPStatus(Internal("")) != 500 {
		t.Fatalf("50000 应映射 HTTP 500: %d", HTTPStatus(Internal("")))
	}
	if HTTPStatus(errors.New("raw")) != 500 {
		t.Fatalf("非业务错误应映射 HTTP 500: %d", HTTPStatus(errors.New("raw")))
	}
}
