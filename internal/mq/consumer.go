package mq

import (
	"context"
	"errors"
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

	"photography-server/internal/pkg/logger"
)

// 消费并发模型（2026-09-21 核实 nats.go v1.53.1 源码后整理）：
//
// nats.go 对**每个订阅只创建一个** waitForMsgs goroutine 并串行回调；Chan 订阅由读循环
// 直接投递、无独立 goroutine。故 goroutine 数上界 = 订阅数，与消息峰值无关 —— 不存在
// 「每条消息起一个 goroutine 导致爆炸」。峰值下的真实风险是：
//
//	· 单订阅串行 → 队头阻塞（同主题慢消息拖住整条主题，但不同主题互不影响）；
//	· 订阅 pending 队列（默认 65536 条 / 64MB）写满后由库报 SlowConsumer 并**丢弃**消息。
//
// 因此这里**不引入全局 worker pool**：并发 >1 会破坏同主题消息顺序，而订单状态流转、
// 支付回调都依赖顺序。需要提吞吐时应按业务键（如 order_id）分片，而非无脑加并发。
// 本文件在峰值下的加固点：JetStream MaxAckPending 服务端限流 + Pull 持续拉取 + panic 就地兜住。
const (
	// pullBatchSize 单次 Fetch 条数；配合 MaxAckPending，避免一次拉入过多在途消息。
	pullBatchSize = 32
	// maxAckPending JetStream 在途未确认上限：服务端据此限速，消费端内存有界。
	maxAckPending = 64
	// pullMaxWait Fetch 空队列等待；到点返回以便检查停机信号。
	pullMaxWait = 500 * time.Millisecond
	// pullErrorBackoff Fetch 失败后的退避，避免错误风暴空转。
	pullErrorBackoff = 1 * time.Second
)

// natsHandler 业务消息处理器：ctx 为续接链路后的上下文（从消息 Header 抽取的父 span 派生），
// 业务内所有 SQL（repository 已 ctx 贯穿 WithContext）自动挂到同一 trace。
type natsHandler func(ctx context.Context, msg *nats.Msg)

// Consumer NATS 消费者
type Consumer struct {
	nc       *nats.Conn
	js       nats.JetStreamContext
	tracer   trace.Tracer
	subs     []*nats.Subscription
	pullSubs []*pullSub
	stopCh   chan struct{}
	stopOnce sync.Once
	wg       sync.WaitGroup
}

// pullSub 封装 Pull 订阅及其处理函数
type pullSub struct {
	sub     *nats.Subscription
	handler nats.MsgHandler
}

// New 创建消费者实例。tracer 由组合根注入（#40；链路未启用时为 nil，traced 退化为直调）。
func New(nc *nats.Conn, tracer trace.Tracer) *Consumer {
	c := &Consumer{
		nc:     nc,
		tracer: tracer,
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

// Start 启动所有消费者。
//
// 返回非 nil 表示至少有一个核心订阅未能建立——这正是「服务看起来正常、但异步流程
// （订单状态变更、通知推送、支付回调）全部停摆」的静默故障场景（#45.4）。
// 调用方必须把它当作致命错误处理，而不是仅记日志继续启动。
//
// 说明：JetStream 不可用（连的是普通 NATS）时按既定策略降级为告警，不计入错误——
// 该部署形态下持久化订阅本就无法建立，不应阻断启动；但若 JS 可用而个别订阅失败，
// 属于真实故障，必须上报。
func (c *Consumer) Start() error {
	if c.nc == nil {
		return errors.New("nats 未连接，消费者无法启动（异步流程将全部停摆）")
	}

	var failures []error

	// ===== 非持久化消息消费 =====
	failures = append(failures, c.subscribe("test.msg", handleTestMsg))
	failures = append(failures, c.subscribe("order.status.change", handleOrderStatusChange))
	failures = append(failures, c.subscribe("notification.push", handleNotificationPush))

	// ===== JetStream Push 消费（回调自动处理） =====
	failures = append(failures, c.jsSubscribe("photography.test.persistent", handleTestPersistent))
	failures = append(failures, c.jsSubscribe("photography.order.created.persistent", handleOrderCreatedPersistent))
	failures = append(failures, c.jsSubscribe("photography.payment.callback.persistent", handlePaymentCallbackPersistent))

	// ===== JetStream Pull 消费（循环拉取） =====
	failures = append(failures, c.jsPullSubscribe("photography.test.pull", handleTestPull))
	failures = append(failures, c.jsPullSubscribe("photography.order.pull", handleOrderPull))

	// 启动 Pull 消费循环
	c.startPullLoop()

	logger.Infof("nats consumers started, push: %d, pull: %d", len(c.subs), len(c.pullSubs))

	if err := errors.Join(failures...); err != nil {
		return fmt.Errorf("nats 部分消费者启动失败（异步流程不完整）: %w", err)
	}
	return nil
}

// Stop 停止所有订阅。幂等：重复调用不会 panic（close 已关闭的 channel 会 panic）。
func (c *Consumer) Stop() {
	c.stopOnce.Do(func() {
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
	})
}

// ======================== 订阅方法 ========================

// subscribe 非持久化订阅，失败返回 error（由 Start 汇总上报）
func (c *Consumer) subscribe(subject string, handler natsHandler) error {
	sub, err := c.nc.Subscribe(subject, c.traced(subject, handler, false))
	if err != nil {
		logger.Errorf("nats subscribe [%s] failed: %v", subject, err)
		return fmt.Errorf("subscribe %s: %w", subject, err)
	}
	c.subs = append(c.subs, sub)
	logger.Infof("nats subscribed: %s", subject)
	return nil
}

// jsSubscribe JetStream Push 订阅（回调自动处理，服务启动一次即可）
func (c *Consumer) jsSubscribe(subject string, handler natsHandler) error {
	if c.js == nil {
		logger.Warnf("jetStream not available, skip push subscribe [%s]", subject)
		return nil
	}
	durable := durableName(subject)
	sub, err := c.js.Subscribe(subject, c.traced(subject, handler, true),
		nats.Durable(durable),
		nats.ManualAck(),
		nats.DeliverAll(),
		nats.MaxDeliver(3),
		nats.AckWait(30*time.Second),
		nats.MaxAckPending(maxAckPending),
	)
	if err != nil {
		logger.Errorf("jetStream push subscribe [%s] failed: %v", subject, err)
		return fmt.Errorf("js push subscribe %s: %w", subject, err)
	}
	c.subs = append(c.subs, sub)
	logger.Infof("jetStream push subscribed: %s (durable: %s)", subject, durable)
	return nil
}

// jsPullSubscribe JetStream Pull 订阅（注册到列表，由 startPullLoop 循环拉取）
func (c *Consumer) jsPullSubscribe(subject string, handler natsHandler) error {
	if c.js == nil {
		logger.Warnf("jetStream not available, skip pull subscribe [%s]", subject)
		return nil
	}
	durable := durableName(subject)
	sub, err := c.js.PullSubscribe(subject, durable,
		nats.DeliverAll(),
		nats.MaxDeliver(3),
		nats.AckWait(30*time.Second),
		nats.MaxAckPending(maxAckPending),
	)
	if err != nil {
		logger.Errorf("jetStream pull subscribe [%s] failed: %v", subject, err)
		return fmt.Errorf("js pull subscribe %s: %w", subject, err)
	}
	c.pullSubs = append(c.pullSubs, &pullSub{sub: sub, handler: c.traced(subject, handler, true)})
	logger.Infof("jetStream pull subscribed: %s (durable: %s)", subject, durable)
	return nil
}

// startPullLoop 启动 Pull 消费循环：每个 Pull 订阅一个独立 goroutine。
//
// 旧实现是全局 1s ticker（每主题每秒最多 10 条），吞吐被人为压死且在途不可控；
// 现由 Fetch 阻塞驱动**持续拉取**，处理不过来时 MaxAckPending 在服务端限流，消费端内存有界。
// 批内仍顺序处理，保留同主题消息顺序（订单/支付类主题依赖顺序）。
func (c *Consumer) startPullLoop() {
	for _, ps := range c.pullSubs {
		c.wg.Add(1)
		go c.pullLoop(ps)
	}
}

// pullLoop 单个 Pull 订阅的消费循环（顺序处理，保留顺序语义）
func (c *Consumer) pullLoop(ps *pullSub) {
	defer c.wg.Done()
	for {
		select {
		case <-c.stopCh:
			return
		default:
		}

		msgs, err := ps.sub.Fetch(pullBatchSize, nats.MaxWait(pullMaxWait))
		if err != nil {
			if errors.Is(err, nats.ErrTimeout) {
				continue // 空队列属正常，回到循环顶检查停机信号
			}
			select {
			case <-c.stopCh:
				return
			case <-time.After(pullErrorBackoff):
			}
			logger.Errorf("pull fetch [%s] failed: %v", ps.sub.Subject, err)
			continue
		}
		for _, msg := range msgs {
			ps.handler(msg)
		}
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
func (c *Consumer) traced(subject string, handler natsHandler, jetStream bool) nats.MsgHandler {
	return func(msg *nats.Msg) {
		start := time.Now()
		tr := c.tracer
		if tr == nil {
			defer func() {
				if r := recover(); r != nil {
					c.onPanic(subject, msg, jetStream, r)
				}
				logger.Infof("[nats:%s] done, cost=%dms", subject, time.Since(start).Milliseconds())
			}()
			handler(context.Background(), msg)
			return
		}
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
				c.onPanic(subject, msg, jetStream, r)
			}
			span.End()
			logger.Infof("[nats:%s] done, trace=%s, cost=%dms", subject, tid, time.Since(start).Milliseconds())
		}()
		handler(ctx, msg)
	}
}

// onPanic 统一处理 handler panic。
//
// 旧实现 recover 后**重新 panic**，会把 nats 的 waitForMsgs / Fetch goroutine 打崩 ——
// 单条坏消息即可导致整个服务重启，且 JetStream 消息因未 ack 被反复重投。
// 现改为就地兜住并记录；JetStream 消息额外 Nak，按 MaxDeliver(3) 重投。
func (c *Consumer) onPanic(subject string, msg *nats.Msg, jetStream bool, r interface{}) {
	logger.Errorf("[nats:%s] handler panic recovered: %v", subject, r)
	if jetStream && msg != nil && msg.Reply != "" {
		if err := msg.Nak(); err != nil {
			logger.Warnf("[nats:%s] nak after panic failed: %v", subject, err)
		}
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
