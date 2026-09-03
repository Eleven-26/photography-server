package mq

import (
	"strings"
	"time"

	"github.com/nats-io/nats.go"

	"photography-server/internal/pkg/logger"
)

// Consumer NATS 消费者
type Consumer struct {
	nc       *nats.Conn
	js       nats.JetStreamContext
	subs     []*nats.Subscription
	pullSubs []*nats.Subscription
}

// New 创建消费者实例
func New(nc *nats.Conn) *Consumer {
	c := &Consumer{nc: nc, subs: make([]*nats.Subscription, 0)}
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

// Start 启动所有消费者（Push + Pull）
func (c *Consumer) Start() {
	if c.nc == nil {
		logger.Warnf("nats not connected, consumer skipped")
		return
	}

	// ===== 非持久化消息消费 =====
	c.subscribe("test.msg", handleTestMsg)
	c.subscribe("order.status.change", handleOrderStatusChange)
	c.subscribe("notification.push", handleNotificationPush)

	// ===== JetStream Push 消费（服务启动一次，自动接收） =====
	c.jsSubscribe("photography.test.persistent", handleTestPersistent)
	c.jsSubscribe("photography.order.created.persistent", handleOrderCreatedPersistent)
	c.jsSubscribe("photography.payment.callback.persistent", handlePaymentCallbackPersistent)

	// ===== JetStream Pull 消费（按需拉取） =====
	c.jsPullSubscribe("photography.test.pull", handleTestPull)
	c.jsPullSubscribe("photography.order.pull", handleOrderPull)

	logger.Infof("nats consumers started, push: %d, pull: %d", len(c.subs), len(c.pullSubs))
}

// Stop 停止所有订阅
func (c *Consumer) Stop() {
	for _, sub := range c.subs {
		if sub.IsValid() {
			sub.Unsubscribe()
		}
	}
	for _, sub := range c.pullSubs {
		if sub.IsValid() {
			sub.Unsubscribe()
		}
	}
	c.subs = nil
	c.pullSubs = nil
	logger.Infof("nats consumers stopped")
}

// PullMessages 拉取 Pull 订阅的消息（外部调用）
func (c *Consumer) PullMessages(subject string, batchSize int) ([]*nats.Msg, error) {
	for _, sub := range c.pullSubs {
		if sub.Subject == subject {
			return sub.Fetch(batchSize)
		}
	}
	return nil, nats.ErrBadSubscription
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

// jsSubscribe JetStream Push 订阅（服务启动后自动接收消息）
// 服务只需启动一次，后续有消息自动推送
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

// jsPullSubscribe JetStream Pull 订阅（按需拉取消息）
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
	c.pullSubs = append(c.pullSubs, sub)
	logger.Infof("jetStream pull subscribed: %s (durable: %s)", subject, durable)
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
