package routes

import (
	"photography-server/internal/domain"
	"photography-server/internal/presentation/controller"
)

// orderFlow 订单 / 订单加项 / 改期。
//
// 订单金额无独立修改接口——改价经加项同事务重算，走 order:update。
// 加项（妆造/时效/服务/精修）：增删改同事务重算订单金额，归订单编辑权限。
// 改期：PC 不直接改订单拍摄日期（会漏掉档期锁重排），统一走改期单链路；
// apply = 发起改期（销售/店长），audit = 审批改期（店长），二者权限点分离。
func orderFlow(ctl *controller.Controller) []Route {
	return []Route{
		{Path: "/order/create", Perm: domain.PermOrderCreate, Handler: ctl.OrderCreate},
		{Path: "/order/list", Perm: domain.PermOrderView, Handler: ctl.OrderList},
		{Path: "/order/detail/:id", Perm: domain.PermOrderView, Handler: ctl.OrderDetail},
		{Path: "/order/update/:id", Perm: domain.PermOrderUpdate, Handler: ctl.OrderUpdate},
		{Path: "/order/status/:id", Perm: domain.PermOrderStatus, Handler: ctl.OrderStatus},
		{Path: "/order/cancel/:id", Perm: domain.PermOrderCancel, Handler: ctl.OrderCancel},
		{Path: "/order/logs/:id", Perm: domain.PermOrderView, Handler: ctl.OrderLogs},

		{Path: "/order/addon/list/:order_id", Perm: domain.PermOrderView, Handler: ctl.OrderAddonList},
		{Path: "/order/addon/create/:order_id", Perm: domain.PermOrderUpdate, Handler: ctl.OrderAddonCreate},
		{Path: "/order/addon/update/:id", Perm: domain.PermOrderUpdate, Handler: ctl.OrderAddonUpdate},
		{Path: "/order/addon/delete/:id", Perm: domain.PermOrderUpdate, Handler: ctl.OrderAddonDelete},

		{Path: "/order/reschedule/list/:order_id", Perm: domain.PermOrderView, Handler: ctl.OrderRescheduleList},
		{Path: "/order/reschedule/apply/:order_id", Perm: domain.PermOrderReschedule, Handler: ctl.OrderRescheduleApply},
		{Path: "/order/reschedule/audit/:id", Perm: domain.PermOrderRescheduleAudit, Handler: ctl.OrderRescheduleAudit},
	}
}
