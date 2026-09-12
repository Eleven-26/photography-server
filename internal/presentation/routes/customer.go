package routes

import (
	"photography-server/internal/domain"
	"photography-server/internal/presentation/controller"
)

// customerLeadQuote 客户 / 线索 / 报价。
func customerLeadQuote(ctl *controller.Controller) []Route {
	return []Route{
		// 客户（stats / orders 为客户的只读派生视图，与 list/detail 同权限）
		{Path: "/customer/list", Perm: domain.PermCustomerView, Handler: ctl.CustomerList},
		{Path: "/customer/detail/:id", Perm: domain.PermCustomerView, Handler: ctl.CustomerDetail},
		{Path: "/customer/create", Perm: domain.PermCustomerCreate, Handler: ctl.CustomerCreate},
		{Path: "/customer/update/:id", Perm: domain.PermCustomerUpdate, Handler: ctl.CustomerUpdate},
		{Path: "/customer/delete/:id", Perm: domain.PermCustomerDelete, Handler: ctl.CustomerDelete},
		{Path: "/customer/stats", Perm: domain.PermCustomerView, Handler: ctl.CustomerStats},
		{Path: "/customer/orders/:id", Perm: domain.PermCustomerView, Handler: ctl.CustomerOrders},

		// 线索（沟通记录与 AI 简报读接口归 lead:view，写入归 lead:update；
		// 变更归属人属"分配"语义，路由级无法区分，由 service 层追加 lead:assign 校验）
		{Path: "/lead/list", Perm: domain.PermLeadView, Handler: ctl.LeadList},
		{Path: "/lead/detail/:id", Perm: domain.PermLeadView, Handler: ctl.LeadDetail},
		{Path: "/lead/create", Perm: domain.PermLeadCreate, Handler: ctl.LeadCreate},
		{Path: "/lead/update/:id", Perm: domain.PermLeadUpdate, Handler: ctl.LeadUpdate},
		{Path: "/lead/delete/:id", Perm: domain.PermLeadDelete, Handler: ctl.LeadDelete},
		{Path: "/lead/follow/:id", Perm: domain.PermLeadUpdate, Handler: ctl.LeadFollow},
		{Path: "/lead/convert/:id", Perm: domain.PermLeadConvert, Handler: ctl.LeadConvert},
		{Path: "/lead/messages/:id", Perm: domain.PermLeadView, Handler: ctl.LeadMessages},
		{Path: "/lead/message/send/:id", Perm: domain.PermLeadUpdate, Handler: ctl.LeadMessageSend},
		{Path: "/lead/brief/list/:lead_id", Perm: domain.PermLeadView, Handler: ctl.LeadBriefList},
		{Path: "/lead/brief/generate/:lead_id", Perm: domain.PermLeadUpdate, Handler: ctl.LeadBriefGenerate},

		// 报价（status 为接受/拒绝/成交/撤回的状态流转，归 quote:update；
		// 报价无独立审批接口，quote:audit 已随 2026-09-11 权限点清理移除）
		{Path: "/quote/create/:lead_id", Perm: domain.PermQuoteCreate, Handler: ctl.QuoteCreate},
		{Path: "/quote/list/:lead_id", Perm: domain.PermQuoteView, Handler: ctl.QuoteList},
		{Path: "/quote/status/:id", Perm: domain.PermQuoteUpdate, Handler: ctl.QuoteStatus},
	}
}
