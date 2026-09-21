package mq

import (
	"context"
	"testing"

	"github.com/nats-io/nats.go"
)

// TestTracedRecoversPanic handler panic 不得打崩进程，也不得向上抛出（交还 nats 会崩）。
// 回归的是旧实现「recover 后重新 panic」的缺陷：单条坏消息即可拖垮整个服务。
func TestTracedRecoversPanic(t *testing.T) {
	c := New(nil, nil)
	called := false
	h := c.traced("test.subject", func(ctx context.Context, msg *nats.Msg) {
		called = true
		panic("boom")
	}, false)

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panic 未被兜住（会崩服务）: %v", r)
		}
	}()
	h(&nats.Msg{Subject: "test.subject"})

	if !called {
		t.Fatal("handler 未被调用")
	}
}
