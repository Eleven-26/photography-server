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

	// 示例：订单状态变更消费
	c.subscribe("order.status.change", handleOrderStatusChange)

	// 示例：通知消息消费
	c.subscribe("notification.push", handleNotificationPush)

	// 示例：测试消费
	c.subscribe("test.msg", handleTestMsg)

	logger.Infof("nats consumers started")
}

// subscribe 订阅 subject 并注册回调
func (c *Consumer) subscribe(subject string, handler nats.MsgHandler) {
	_, err := c.nc.Subscribe(subject, handler)
	if err != nil {
		logger.Errorf("nats subscribe [%s] failed: %v", subject, err)
		return
	}
	logger.Infof("nats subscribed: %s", subject)
}

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
