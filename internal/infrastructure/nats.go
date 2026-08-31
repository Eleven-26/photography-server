package infrastructure

import (
	"sync"
	"time"

	"github.com/nats-io/nats.go"

	"photography-server/internal/config"
)

var (
	natsInstance *nats.Conn
	natsOnce     sync.Once
	natsErr      error
)

// InitNATS 初始化 NATS 单例
func InitNATS(c *config.NATS) error {
	natsOnce.Do(func() {
		natsInstance, natsErr = nats.Connect(c.URL,
			nats.MaxReconnects(10),
			nats.ReconnectWait(2*time.Second),
			nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
				// 断连时由 nats 自动重连
			}),
			nats.ErrorHandler(func(nc *nats.Conn, sub *nats.Subscription, err error) {
				// 可替换为 logger
			}),
		)
	})
	return natsErr
}

// NATS 获取 NATS 单例
func NATS() *nats.Conn {
	return natsInstance
}
