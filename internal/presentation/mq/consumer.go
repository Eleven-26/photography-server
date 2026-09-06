package mq

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"

	"photography-server/internal/infrastructure"
	"photography-server/internal/pkg/logger"
)

// natsHandler 业务消息处理器：ctx 为续接链路后的上下文（从消息 Header 抽取的父 span 派生），
// 业务内所有 SQL（repository 已 ctx 贯穿 WithContext）自动挂到同一 trace。
type natsHandler func(ctx context.Context, msg *nats.Msg)

// Consumer NATS 消费者
type Consumer struct {
	nc       *nats.Conn
	js       nats.JetStreamContext
	subs     []*nats.Subscription
	pullSubs []*pullSub
	stopCh   chan struct{}
	wg       sync.WaitGroup
}

// pullSub 封装 Pull 订阅及其处理函数
type pullSub struct {
	sub     *nats.Subscription
	handler nats.MsgHandler
}

// New 创建消费者实例
func New(nc *nats.Conn) *Consumer {
	c := &Consumer{
		nc:     nc,
		subs:   make([]*nats.Subscription, 0),
		stopCh: make(chan struct{}),
	}
	if nc != nil {
		js, err := nc.JetStream()
		if err != nil {
			logger.Warnf("jetStream not available: %v", err)
		} else {
			c.js = js
		}
	}
	return c
}

// Start 启动所有消费者
func (c *Consumer) Start() {
	if c.nc == nil {
		logger.Warnf("nats not connected, consumer skipped")
		return
	}

	// ===== 非持久化消息消费 =====
	c.subscribe("test.msg", handleTestMsg)
	c.subscribe("order.status.change", handleOrderStatusChange)
	c.subscribe("notification.push", handleNotificationPush)

	// ===== JetStream Push 消费（回调自动处理） =====
	c.jsSubscribe("photography.test.persistent", handleTestPersistent)
	c.jsSubscribe("photography.order.created.persistent", handleOrderCreatedPersistent)
	c.jsSubscribe("photography.payment.callback.persistent", handlePaymentCallbackPersistent)

	// ===== JetStream Pull 消费（循环拉取） =====
	c.jsPullSubscribe("photography.test.pull", handleTestPull)
	c.jsPullSubscribe("photography.order.pull", handleOrderPull)

	// 启动 Pull 消费循环
	c.startPullLoop()

	logger.Infof("nats consumers started, push: %d, pull: %d", len(c.subs), len(c.pullSubs))
}

// Stop 停止所有订阅
func (c *Consumer) Stop() {
	close(c.stopCh)
	c.wg.Wait()

	for _, sub := range c.subs {
		if sub.IsValid() {
			sub.Unsubscribe()
		}
	}
	for _, ps := range c.pullSubs {
		if ps.sub.IsValid() {
			ps.sub.Unsubscribe()
		}
	}
	c.subs = nil
	c.pullSubs = nil
	logger.Infof("nats consumers stopped")
}

// ======================== 订阅方法 ========================

// subscribe 非持久化订阅
func (c *Consumer) subscribe(subject string, handler natsHandler) {
	sub, err := c.nc.Subscribe(subject, traced(subject, handler))
	if err != nil {
		logger.Errorf("nats subscribe [%s] failed: %v", subject, err)
		return
	}
	c.subs = append(c.subs, sub)
	logger.Infof("nats subscribed: %s", subject)
}

// jsSubscribe JetStream Push 订阅（回调自动处理，服务启动一次即可）
func (c *Consumer) jsSubscribe(subject string, handler natsHandler) {
	if c.js == nil {
		logger.Warnf("jetStream not available, skip push subscribe [%s]", subject)
		return
	}
	durable := durableName(subject)
	sub, err := c.js.Subscribe(subject, traced(subject, handler),
		nats.Durable(durable),
		nats.ManualAck(),
		nats.DeliverAll(),
		nats.MaxDeliver(3),
		nats.AckWait(30*time.Second),
	)
	if err != nil {
		logger.Errorf("jetStream push subscribe [%s] failed: %v", subject, err)
		return
	}
	c.subs = append(c.subs, sub)
	logger.Infof("jetStream push subscribed: %s (durable: %s)", subject, durable)
}

// jsPullSubscribe JetStream Pull 订阅（注册到列表，由 startPullLoop 循环拉取）
func (c *Consumer) jsPullSubscribe(subject string, handler natsHandler) {
	if c.js == nil {
		logger.Warnf("jetStream not available, skip pull subscribe [%s]", subject)
		return
	}
	durable := durableName(subject)
	sub, err := c.js.PullSubscribe(subject, durable,
		nats.DeliverAll(),
		nats.MaxDeliver(3),
		nats.AckWait(30*time.Second),
	)
	if err != nil {
		logger.Errorf("jetStream pull subscribe [%s] failed: %v", subject, err)
		return
	}
	c.pullSubs = append(c.pullSubs, &pullSub{sub: sub, handler: traced(subject, handler)})
	logger.Infof("jetStream pull subscribed: %s (durable: %s)", subject, durable)
}

// startPullLoop 启动 Pull 消费循环（每秒拉取一次）
func (c *Consumer) startPullLoop() {
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-c.stopCh:
				return
			case <-ticker.C:
				for _, ps := range c.pullSubs {
					c.fetchMessages(ps)
				}
			}
		}
	}()
}

// fetchMessages 拉取并处理一批消息
func (c *Consumer) fetchMessages(ps *pullSub) {
	msgs, err := ps.sub.Fetch(10, nats.MaxWait(500*time.Millisecond))
	if err != nil {
		// 超时是正常的，说明没有新消息
		if err == nats.ErrTimeout {
			return
		}
		logger.Errorf("pull fetch [%s] failed: %v", ps.sub.Subject, err)
		return
	}
	for _, msg := range msgs {
		ps.handler(msg)
	}
}

func durableName(subject string) string {
	return strings.ReplaceAll(subject, ".", "_")
}

// ======================== 链路透传包装 ========================

// traced 消费链路包装（Jaeger/OTel 通道）：从消息 Header 抽取生产端注入的 W3C TraceContext 作为
// 父上下文，在父上下文上创建"处理 span"续接同一 trace（生产→消费完整链路）。
// 消息无 traceparent（手动测试/旧消息）时退化为独立根 span，保证每次处理都有可查链路。
// 追踪未启用时原样调用业务 handler，零开销。
// 注意：SkyWalking-go（native）通道的消息 Header 无 OTel traceparent（agent 未注入），
// 该通道下 MQ 透传需按 sw8 header 格式手动接入，为 P1 待办。
func traced(subject string, handler natsHandler) nats.MsgHandler {
	return func(msg *nats.Msg) {
		tr := infrastructure.JaegerTracer()
		if tr == nil {
			handler(context.Background(), msg)
			return
		}
		start := time.Now()
		parent := otel.GetTextMapPropagator().Extract(context.Background(), propagation.HeaderCarrier(msg.Header))
		ctx, span := tr.Start(parent, "nats."+subject,
			trace.WithAttributes(
				attribute.String("messaging.system", "nats"),
				attribute.String("messaging.destination", msg.Subject),
				attribute.String("messaging.operation", "process"),
			),
		)
		tid := span.SpanContext().TraceID().String()
		logger.Infof("[nats:%s] start, trace=%s", subject, tid)
		defer func() {
			if r := recover(); r != nil {
				span.RecordError(fmt.Errorf("handler panic: %v", r))
				span.SetStatus(codes.Error, "handler panic")
				span.End()
				logger.Errorf("[nats:%s] panic, trace=%s, err=%v", subject, tid, r)
				panic(r) // 交还 nats 库 recover，不吞异常
			}
			span.End()
			logger.Infof("[nats:%s] done, trace=%s, cost=%dms", subject, tid, time.Since(start).Milliseconds())
		}()
		handler(ctx, msg)
	}
}

// ======================== 非持久化消息处理 ========================

func handleTestMsg(ctx context.Context, msg *nats.Msg) {
	logger.Infof("[TestConsumer] subject: %s, data: %s", msg.Subject, string(msg.Data))
}

func handleOrderStatusChange(ctx context.Context, msg *nats.Msg) {
	logger.Infof("[OrderStatusChange] subject: %s, data: %s", msg.Subject, string(msg.Data))
}

func handleNotificationPush(ctx context.Context, msg *nats.Msg) {
	logger.Infof("[NotificationPush] subject: %s, data: %s", msg.Subject, string(msg.Data))
}

// ======================== JetStream Push 消息处理 ========================

func handleTestPersistent(ctx context.Context, msg *nats.Msg) {
	logger.Infof("[TestPush] subject: %s, data: %s", msg.Subject, string(msg.Data))
	msg.Ack()
}

func handleOrderCreatedPersistent(ctx context.Context, msg *nats.Msg) {
	logger.Infof("[OrderCreatedPush] subject: %s, data: %s", msg.Subject, string(msg.Data))
	msg.Ack()
}

func handlePaymentCallbackPersistent(ctx context.Context, msg *nats.Msg) {
	logger.Infof("[PaymentCallbackPush] subject: %s, data: %s", msg.Subject, string(msg.Data))
	msg.Ack()
}

// ======================== JetStream Pull 消息处理 ========================

func handleTestPull(ctx context.Context, msg *nats.Msg) {
	logger.Infof("[TestPull] subject: %s, data: %s", msg.Subject, string(msg.Data))
	msg.Ack()
}

func handleOrderPull(ctx context.Context, msg *nats.Msg) {
	logger.Infof("[OrderPull] subject: %s, data: %s", msg.Subject, string(msg.Data))
	msg.Ack()
}
