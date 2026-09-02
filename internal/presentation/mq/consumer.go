package mq

import (
	"github.com/nats-io/nats.go"

	"photography-server/internal/pkg/logger"
)

// Consumer NATS 消费者
type Consumer struct {
	nc *nats.Conn
}

// New 创建消费者实例
func New(nc *nats.Conn) *Consumer {
	return &Consumer{nc: nc}
}

// Start 启动所有消费者订阅
func (c *Consumer) Start() {
	if c.nc == nil {
		logger.Warnf("nats not connected, consumer skipped")
		return
	}

	// ===== 非持久化消息消费 =====

	// 测试消费
	c.subscribe("test.msg", handleTestMsg)

	// 订单状态变更消费
	c.subscribe("order.status.change", handleOrderStatusChange)

	// 通知消息消费
	c.subscribe("notification.push", handleNotificationPush)

	// ===== 持久化消息消费（JetStream） =====

	// 测试持久化消费
	c.jsSubscribe("test.persistent", handleTestPersistent)

	// 订单创建持久化消费
	c.jsSubscribe("order.created.persistent", handleOrderCreatedPersistent)

	// 支付回调持久化消费
	c.jsSubscribe("payment.callback.persistent", handlePaymentCallbackPersistent)

	logger.Infof("nats consumers started")
}

// subscribe 订阅非持久化消息
func (c *Consumer) subscribe(subject string, handler nats.MsgHandler) {
	_, err := c.nc.Subscribe(subject, handler)
	if err != nil {
		logger.Errorf("nats subscribe [%s] failed: %v", subject, err)
		return
	}
	logger.Infof("nats subscribed: %s", subject)
}

// jsSubscribe 订阅持久化消息（JetStream）
func (c *Consumer) jsSubscribe(subject string, handler nats.MsgHandler) {
	js, err := c.nc.JetStream()
	if err != nil {
		logger.Warnf("jetStream not available, skip persistent subscribe [%s]: %v", subject, err)
		return
	}
	_, err = js.Subscribe(subject, handler)
	if err != nil {
		logger.Errorf("jetStream subscribe [%s] failed: %v", subject, err)
		return
	}
	logger.Infof("jetStream subscribed: %s", subject)
}

// ======================== 非持久化消息处理 ========================

// handleTestMsg 测试消息消费
func handleTestMsg(msg *nats.Msg) {
	logger.Infof("[TestConsumer] subject: %s, data: %s", msg.Subject, string(msg.Data))
}

// handleOrderStatusChange 订单状态变更消费
func handleOrderStatusChange(msg *nats.Msg) {
	logger.Infof("[OrderStatusChange] subject: %s, data: %s", msg.Subject, string(msg.Data))
	// TODO: 实现订单状态变更后的通知逻辑
	// 例如：更新通知表、推送站内信等
}

// handleNotificationPush 通知推送消费
func handleNotificationPush(msg *nats.Msg) {
	logger.Infof("[NotificationPush] subject: %s, data: %s", msg.Subject, string(msg.Data))
	// TODO: 实现通知推送逻辑
	// 例如：WebSocket 推送、公众号模板消息等
}

// ======================== 持久化消息处理 ========================

// handleTestPersistent 测试持久化消费
func handleTestPersistent(msg *nats.Msg) {
	logger.Infof("[TestPersistent] subject: %s, data: %s", msg.Subject, string(msg.Data))
	// ACK 消息
	if err := msg.Ack(); err != nil {
		logger.Errorf("[TestPersistent] ack failed: %v", err)
	}
}

// handleOrderCreatedPersistent 订单创建持久化消费
func handleOrderCreatedPersistent(msg *nats.Msg) {
	logger.Infof("[OrderCreatedPersistent] subject: %s, data: %s", msg.Subject, string(msg.Data))
	// TODO: 订单创建后的持久化处理
	// 例如：发送确认通知、记录审计日志等
	if err := msg.Ack(); err != nil {
		logger.Errorf("[OrderCreatedPersistent] ack failed: %v", err)
	}
}

// handlePaymentCallbackPersistent 支付回调持久化消费
func handlePaymentCallbackPersistent(msg *nats.Msg) {
	logger.Infof("[PaymentCallbackPersistent] subject: %s, data: %s", msg.Subject, string(msg.Data))
	// TODO: 支付回调的持久化处理
	// 例如：更新订单状态、记录支付流水等
	if err := msg.Ack(); err != nil {
		logger.Errorf("[PaymentCallbackPersistent] ack failed: %v", err)
	}
}
