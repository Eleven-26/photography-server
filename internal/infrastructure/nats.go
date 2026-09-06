package infrastructure

import (
	"context"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"

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

// traceMsg 构造携带 W3C TraceContext 的消息：把 ctx 中的 span 上下文注入消息 Header，
// 使消费端能抽取同一 trace 续接链路（Jaeger/OTel 通道的生产→消费完整透传）。
// ctx 无有效 span（如链路未启用 / SkyWalking-go native 通道——agent 的链路上下文不在 OTel ctx 中）时
// Inject 为空操作，Header 为空，消息行为与普通消息一致；native 通道的 MQ 透传（sw8 header）为 P1 待办。
func (c *NatsClient) traceMsg(ctx context.Context, subject string, data []byte) *nats.Msg {
	hdr := make(nats.Header)
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(hdr))
	return &nats.Msg{Subject: subject, Header: hdr, Data: data}
}

// Publish 非持久化发布（fire-and-forget），透传请求链路到消息
func (c *NatsClient) Publish(ctx context.Context, subject string, data []byte) error {
	if c == nil || c.nc == nil {
		return nats.ErrConnectionClosed
	}
	return c.nc.PublishMsg(c.traceMsg(ctx, subject, data))
}

// PublishSync 非持久化同步发布，透传请求链路到消息
func (c *NatsClient) PublishSync(ctx context.Context, subject string, data []byte) error {
	if c == nil || c.nc == nil {
		return nats.ErrConnectionClosed
	}
	return c.nc.PublishMsg(c.traceMsg(ctx, subject, data))
}

// PublishPersistent 持久化发布（通过 JetStream，自动创建 Stream），透传请求链路到消息
func (c *NatsClient) PublishPersistent(ctx context.Context, subject string, data []byte) (*nats.PubAck, error) {
	if c == nil || c.js == nil {
		return nil, nats.ErrJetStreamNotEnabled
	}
	ack, err := c.js.PublishMsg(c.traceMsg(ctx, subject, data))
	if err != nil {
		// 可能 stream 不存在，尝试重建后重试
		logger.Warnf("jetStream publish [%s] failed, try recreate stream: %v", subject, err)
		c.ensureStream(defaultStream, ">")
		ack, err = c.js.PublishMsg(c.traceMsg(ctx, subject, data))
	}
	return ack, err
}

// Request 请求-响应模式，透传请求链路到消息
func (c *NatsClient) Request(ctx context.Context, subject string, data []byte, timeout time.Duration) ([]byte, error) {
	if c == nil || c.nc == nil {
		return nil, nats.ErrConnectionClosed
	}
	msg, err := c.nc.RequestMsg(c.traceMsg(ctx, subject, data), timeout)
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
