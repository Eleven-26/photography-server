package mq

import (
	"strings"
	"sync"
	"time"

	"github.com/nats-io/nats.go"

	"photography-server/internal/pkg/logger"
)

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
func (c *Consumer) subscribe(subject string, handler nats.MsgHandler) {
	sub, err := c.nc.Subscribe(subject, handler)
	if err != nil {
		logger.Errorf("nats subscribe [%s] failed: %v", subject, err)
		return
	}
	c.subs = append(c.subs, sub)
	logger.Infof("nats subscribed: %s", subject)
}

// jsSubscribe JetStream Push 订阅（回调自动处理，服务启动一次即可）
func (c *Consumer) jsSubscribe(subject string, handler nats.MsgHandler) {
	if c.js == nil {
		logger.Warnf("jetStream not available, skip push subscribe [%s]", subject)
		return
	}
	durable := durableName(subject)
	sub, err := c.js.Subscribe(subject, handler,
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
func (c *Consumer) jsPullSubscribe(subject string, handler nats.MsgHandler) {
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
	c.pullSubs = append(c.pullSubs, &pullSub{sub: sub, handler: handler})
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

// ======================== 非持久化消息处理 ========================

func handleTestMsg(msg *nats.Msg) {
	logger.Infof("[TestConsumer] subject: %s, data: %s", msg.Subject, string(msg.Data))
}

func handleOrderStatusChange(msg *nats.Msg) {
	logger.Infof("[OrderStatusChange] subject: %s, data: %s", msg.Subject, string(msg.Data))
}

func handleNotificationPush(msg *nats.Msg) {
	logger.Infof("[NotificationPush] subject: %s, data: %s", msg.Subject, string(msg.Data))
}

// ======================== JetStream Push 消息处理 ========================

func handleTestPersistent(msg *nats.Msg) {
	logger.Infof("[TestPush] subject: %s, data: %s", msg.Subject, string(msg.Data))
	msg.Ack()
}

func handleOrderCreatedPersistent(msg *nats.Msg) {
	logger.Infof("[OrderCreatedPush] subject: %s, data: %s", msg.Subject, string(msg.Data))
	msg.Ack()
}

func handlePaymentCallbackPersistent(msg *nats.Msg) {
	logger.Infof("[PaymentCallbackPush] subject: %s, data: %s", msg.Subject, string(msg.Data))
	msg.Ack()
}

// ======================== JetStream Pull 消息处理 ========================

func handleTestPull(msg *nats.Msg) {
	logger.Infof("[TestPull] subject: %s, data: %s", msg.Subject, string(msg.Data))
	msg.Ack()
}

func handleOrderPull(msg *nats.Msg) {
	logger.Infof("[OrderPull] subject: %s, data: %s", msg.Subject, string(msg.Data))
	msg.Ack()
}
