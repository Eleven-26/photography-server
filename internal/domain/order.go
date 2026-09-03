package domain

import "photography-server/internal/enum"

// orderTransitions 定义允许的订单状态流转（领域状态机，纯数据无副作用，便于单元测试）
var orderTransitions = map[enum.OrderStatus][]enum.OrderStatus{
	enum.OrderStatusPendingDeposit:  {enum.OrderStatusPendingShoot, enum.OrderStatusCancelled},
	enum.OrderStatusPendingShoot:    {enum.OrderStatusShooting, enum.OrderStatusCancelled},
	enum.OrderStatusShooting:        {enum.OrderStatusRetouching, enum.OrderStatusCancelled},
	enum.OrderStatusRetouching:      {enum.OrderStatusPendingDelivery, enum.OrderStatusCancelled},
	enum.OrderStatusPendingDelivery: {enum.OrderStatusCompleted, enum.OrderStatusCancelled},
	enum.OrderStatusCompleted:       {},
	enum.OrderStatusCancelled:       {},
}

// OrderCanTransit 判断订单状态能否从 from 合法流转到 to
func OrderCanTransit(from, to enum.OrderStatus) bool {
	for _, next := range orderTransitions[from] {
		if next == to {
			return true
		}
	}
	return false
}

// OrderAllowedTransitions 返回 from 状态下允许流转的全部目标状态
// （返回副本，防止调用方修改领域状态机内部数据）
func OrderAllowedTransitions(from enum.OrderStatus) []enum.OrderStatus {
	allowed := orderTransitions[from]
	out := make([]enum.OrderStatus, len(allowed))
	copy(out, allowed)
	return out
}
