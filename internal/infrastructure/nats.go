package infrastructure

import (
	"sync"
	"time"

	"github.com/nats-io/nats.go"

	"photography-server/internal/config"
	"photography-server/internal/pkg/logger"
)

const defaultStream = "PHOTOGRAPHY"
const defaultSubject = "photography.>"

// NatsClient NATS 包装客户端（支持持久化/非持久化发布）
type NatsClient struct {
	nc *nats.Conn
	js nats.JetStreamContext
}

var (
	natsConn *nats.Conn
	natsCli  *NatsClient
	natsOnce sync.Once
	natsErr  error
)

// InitNATS 初始化 NATS 单例
func InitNATS(c *config.NATS) error {
	natsOnce.Do(func() {
		natsConn, natsErr = nats.Connect(c.URL,
			nats.MaxReconnects(10),
			nats.ReconnectWait(2*time.Second),
			nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
				logger.Warnf("nats disconnected: %v", err)
			}),
			nats.ReconnectHandler(func(nc *nats.Conn) {
				logger.Infof("nats reconnected to %s", nc.ConnectedUrl())
			}),
			nats.ErrorHandler(func(nc *nats.Conn, sub *nats.Subscription, err error) {
				logger.Errorf("nats error: %v", err)
			}),
		)
		if natsErr == nil {
			natsCli = newNatsClient(natsConn)
			logger.Infof("nats connected to %s", c.URL)
		}
	})
	return natsErr
}

// NATS 获取 NATS 原生连接
func NATS() *nats.Conn {
	return natsConn
}

// GetNatsClient 获取 NATS 包装客户端单例
func GetNatsClient() *NatsClient {
	return natsCli
}

// newNatsClient 创建包装客户端，自动创建默认 Stream
func newNatsClient(nc *nats.Conn) *NatsClient {
	c := &NatsClient{nc: nc}
	if nc != nil {
		js, err := nc.JetStream()
		if err != nil {
			logger.Warnf("nats jetStream not available, persistent mode disabled: %v", err)
		} else {
			c.js = js
			c.ensureStream(defaultStream, defaultSubject)
			logger.Infof("nats jetStream enabled")
		}
	}
	return c
}

// ensureStream 确保 Stream 存在，不存在则创建
func (c *NatsClient) ensureStream(name, subj string) {
	if c == nil || c.js == nil {
		return
	}
	cfg := &nats.StreamConfig{
		Name:      name,
		Subjects:  []string{subj},
		Storage:   nats.FileStorage,
		Retention: nats.LimitsPolicy,
		MaxMsgs:   -1,
		MaxBytes:  -1,
		MaxAge:    24 * time.Hour,
		Replicas:  1,
		NoAck:     false,
	}
	// 查询 stream 是否存在
	info, err := c.js.StreamInfo(name)
	if err != nil {
		// 不存在则创建
		_, err = c.js.AddStream(cfg)
		if err != nil {
			logger.Errorf("create jetStream stream [%s] failed: %v", name, err)
		} else {
			logger.Infof("jetStream stream [%s] created", name)
		}
		return
	}
	// 已存在，检查配置是否一致，不一致则更新
	if info.Config.Subjects == nil || len(info.Config.Subjects) == 0 || info.Config.Subjects[0] != subj {
		_, err = c.js.UpdateStream(cfg)
		if err != nil {
			logger.Errorf("update jetStream stream [%s] failed: %v", name, err)
		} else {
			logger.Infof("jetStream stream [%s] updated", name)
		}
	}
}

// Publish 非持久化发布（fire-and-forget）
func (c *NatsClient) Publish(subject string, data []byte) error {
	if c == nil || c.nc == nil {
		return nats.ErrConnectionClosed
	}
	return c.nc.Publish(subject, data)
}

// PublishSync 非持久化同步发布
func (c *NatsClient) PublishSync(subject string, data []byte) error {
	if c == nil || c.nc == nil {
		return nats.ErrConnectionClosed
	}
	return c.nc.Publish(subject, data)
}

// PublishPersistent 持久化发布（通过 JetStream，自动创建 Stream）
func (c *NatsClient) PublishPersistent(subject string, data []byte) (*nats.PubAck, error) {
	if c == nil || c.js == nil {
		return nil, nats.ErrJetStreamNotEnabled
	}
	ack, err := c.js.Publish(subject, data)
	if err != nil {
		// 可能 stream 不存在，尝试重建后重试
		logger.Warnf("jetStream publish [%s] failed, try recreate stream: %v", subject, err)
		c.ensureStream(defaultStream, ">")
		ack, err = c.js.Publish(subject, data)
	}
	return ack, err
}

// Request 请求-响应模式
func (c *NatsClient) Request(subject string, data []byte, timeout time.Duration) ([]byte, error) {
	if c == nil || c.nc == nil {
		return nil, nats.ErrConnectionClosed
	}
	msg, err := c.nc.Request(subject, data, timeout)
	if err != nil {
		return nil, err
	}
	return msg.Data, nil
}

// Subscribe 订阅消息
func (c *NatsClient) Subscribe(subject string, handler nats.MsgHandler) (*nats.Subscription, error) {
	if c == nil || c.nc == nil {
		return nil, nats.ErrConnectionClosed
	}
	return c.nc.Subscribe(subject, handler)
}

// QueueSubscribe 队列订阅（负载均衡）
func (c *NatsClient) QueueSubscribe(subject, queue string, handler nats.MsgHandler) (*nats.Subscription, error) {
	if c == nil || c.nc == nil {
		return nil, nats.ErrConnectionClosed
	}
	return c.nc.QueueSubscribe(subject, queue, handler)
}

// IsJetStreamEnabled 是否启用 JetStream
func (c *NatsClient) IsJetStreamEnabled() bool {
	return c != nil && c.js != nil
}

// IsConnected 是否已连接
func (c *NatsClient) IsConnected() bool {
	return c != nil && c.nc != nil && c.nc.IsConnected()
}
