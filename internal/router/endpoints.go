package router

import (
	"github.com/gin-gonic/gin"

	"photography-server/internal/domain"
	"photography-server/internal/middleware"
	"photography-server/internal/presentation/controller"
	"photography-server/internal/presentation/endpoint"
	"photography-server/internal/presentation/routes"
	"photography-server/internal/presentation/wechat"
)

// 端暴露面声明（2026-09-12 路由整理）。
//
// 此前「业务路由内容」与「端暴露面」混在一起：管理端路由写在 router.go 的私有函数
// registerCommon 里并硬绑 *controller.Controller，wechat 包引用不到，员工端只能把 24 条
// 同路径路由连同 handler 抄一遍。现在职责分离：
//   - routes 包      → 有哪些业务路由（端无关，唯一实现）
//   - endpoint 包    → 怎么挂（机制）
//   - 本文件         → 每个端暴露哪些（声明）
//
// 端的裁剪决策从"书写顺序"变成下面可读的白名单，护栏测试（routes_test.go）保证
// 白名单里的 Path 确实存在于公共路由表，且跨端 Handler 是同一函数指针。

// pcEndpoint 管理后台：无前缀，员工认证 + 操作日志，全量业务路由。
func pcEndpoint(mw *middleware.Middlewares) endpoint.Endpoint {
	return endpoint.Endpoint{
		Name:        "pc",
		Prefix:      "",
		Middlewares: []gin.HandlerFunc{mw.Auth(), mw.OperationLog()},
	}
}

// miniappEndpoint 小程序管理后台：与 PC 同一份路由表与 handler，仅换前缀。
func miniappEndpoint(mw *middleware.Middlewares) endpoint.Endpoint {
	return endpoint.Endpoint{
		Name:        "miniapp",
		Prefix:      "/miniapp",
		Middlewares: []gin.HandlerFunc{mw.Auth(), mw.OperationLog()},
	}
}

// staffInclude 员工端复用管理端路由表的路径白名单：与 PC 同 Path、同 Handler、同权限点。
//
// 权限点与 PC 端**同源**（domain.Perm 常量、同一张 sys_role_permission），即"员工端与 PC 同权"：
// 某角色能在 PC 做的事，在员工端也放行；反之亦然。若将来需要按端差异化（如禁止员工端导出财务），
// 应另设端维度而非复制权限点。
//
// 明确**不**开放（管理端专属，员工端无对应注册）：全部 role:* / user:* / store:*、
// finance:export、asset 增删改审核、quote、package、calendar/lock 等。
//
// 2026-09-12 决策：**补开 /notification/***。此前员工端缺失通知属遗漏（整理前就未注册，
// 非整理引入）。通知在 service 层已按 receiver_type=1（员工）+ 操作人 UserID 隔离，
// 即"员工本人的通知"，与员工端语义天然匹配；因此直接复用 PC 的 ctl.Notification* 实现
// （同一函数指针），不新建员工端 handler。
//
// 2026-09-12 决策（同日第二批）：**补开 /order/cancel/:id 与 /order/reschedule/apply/:order_id**。
// 小程序确有「发起取消」（pages/refund/review.vue）与「发起改期」（pages/schedule/reschedule.vue）页面，
// 当时"移动端只审批不发起"的口径与前端不符 → 以"小程序可以发起"为准，两条均复用 PC 同一 Handler。
//
//	注意：复用 PC handler 意味着 ApplySource=1（管理端发起）、日志文案为"管理端发起改期"——
//	员工端属管理侧、非客户侧，与 apply_source=2（客户申请）的区分是正确的，文案差异无害。
//
//	  ⚠️ 权限口径：`order:reschedule` 已授 photographer/manager/admin，但 **`order:cancel` 目前
//	  只授 manager/admin**（见 docs/sql 角色种子）。若要让摄影师/销售也能在移动端取消订单，
//	  需另行授权；这涉及业务风险，未擅自改动。
var staffInclude = []string{
	// 订单（复用 PC 端同一 handler 与 service）
	"/order/list",
	"/order/detail/:id",
	"/order/status/:id",
	"/order/logs/:id",
	"/order/create",
	// 取消订单 / 发起改期（小程序可发起；PC 同一 Handler）
	"/order/cancel/:id",
	"/order/reschedule/apply/:order_id",
	// 订单加项（同事务重算订单金额）
	"/order/addon/list/:order_id",
	"/order/addon/create/:order_id",
	"/order/addon/update/:id",
	"/order/addon/delete/:id",
	// 收款（登记；核验见下方 Extra 的 payment/confirm）
	"/payment/create/:order_id",
	"/payment/list/:order_id",
	"/payment/confirm/:id",
	// 交付（读 + 上传样片/成品；创建与确认走 Extra / 未开放）
	// detail 只返回交付单本身、items 才返回文件明细（PC 同口径，:id 均为 order_id）
	"/delivery/detail/:id",
	"/delivery/items/:id",
	"/delivery/upload-samples/:id",
	"/delivery/upload-retouched/:id",
	// 退款（查看；审核见 Extra —— 契约字段与 PC 不同）
	"/refund/list/:order_id",
	// 线索（读 + 沟通）
	"/lead/list",
	"/lead/detail/:id",
	"/lead/messages/:id",
	"/lead/message/send/:id",
	// 客户档案
	"/customer/list",
	"/customer/detail/:id",
	// 通知（员工本人通知；service 按 receiver_type=1 + 操作人 UserID 隔离）
	"/notification/list",
	"/notification/unread-count",
	"/notification/read/:id",
	"/notification/read-all",
}

// staffEndpoint 员工端（挂 /wechat/staff，员工认证 + 操作日志）。
func staffEndpoint(wc *wechat.Controller, ctl *controller.Controller, mw *middleware.Middlewares) endpoint.Endpoint {
	return endpoint.Endpoint{
		Name:        "staff",
		Prefix:      "/wechat/staff",
		Middlewares: []gin.HandlerFunc{mw.StaffAuth(), mw.OperationLog()},
		Include:     staffInclude,
		Extra:       staffExtra(wc, ctl),
	}
}

// staffExtra 员工端 Extra 路由，三类：
//
//  1. **异路径别名**——与 PC 同一 Handler（同一函数指针），仅 URL 是移动端历史路径。
//     保留原路径是为了不破坏小程序前端（在外部仓），同时消除重复实现。
//  2. **真实端差异**——契约或入参确实不同，保留员工端自己的实现（仅此 2 条）。
//  3. **移动端真独有**——员工端独有能力。
func staffExtra(wc *wechat.Controller, ctl *controller.Controller) []routes.Route {
	return []routes.Route{
		// —— 1. 异路径别名：Handler 复用 PC 实现（同一函数指针），仅 Path 不同 ——
		{Path: "/slot-template/list", Perm: domain.PermCalendarView, Handler: ctl.SlotTemplateList},
		{Path: "/slot-template/save", Perm: domain.PermCalendarUpdate, Handler: ctl.SlotTemplateSave},
		{Path: "/slot-template/save/:id", Perm: domain.PermCalendarUpdate, Handler: ctl.SlotTemplateSave},
		{Path: "/slot-template/delete/:id", Perm: domain.PermCalendarUpdate, Handler: ctl.SlotTemplateDelete},
		{Path: "/studio/get", Perm: domain.PermSettingsView, Handler: ctl.StudioGet},
		{Path: "/studio/update", Perm: domain.PermSettingsUpdate, Handler: ctl.StudioUpdate},

		// —— 2. 真实端差异（保留员工端实现，不可合并）——
		// 退款审核：员工端契约是 {"approve": bool}，PC 是 {"approved": *bool} 且必填；
		// 改任一侧都是破坏性变更（两个前端分别依赖各自字段名）。
		{Path: "/refund/audit/:id", Perm: domain.PermRefundAudit, Handler: wc.RefundAudit},
		// 创建交付单：员工端只接 order_id（不传 stage），PC 接完整 DeliveryCreateReq。
		{Path: "/delivery/create/:order_id", Perm: domain.PermDeliveryCreate, Handler: wc.DeliveryCreate},

		// —— 3. 移动端真独有 ——
		{Path: "/overview", Perm: domain.PermDashboardView, Handler: wc.Overview},
		{Path: "/schedule/list", Perm: domain.PermCalendarView, Handler: wc.ScheduleList},
		// 改期：列表与审批（发起走 Include 的 /order/reschedule/apply/:order_id）
		{Path: "/reschedule/list", Perm: domain.PermOrderView, Handler: wc.RescheduleList},
		{Path: "/reschedule/audit/:id", Perm: domain.PermOrderRescheduleAudit, Handler: wc.RescheduleAudit},
		// 线索 AI 简报（读取归 view，生成/发送/确认归 update）
		{Path: "/brief/generate/:lead_id", Perm: domain.PermLeadUpdate, Handler: wc.BriefGenerate},
		{Path: "/brief/list/:lead_id", Perm: domain.PermLeadView, Handler: wc.BriefList},
		{Path: "/brief/send/:id", Perm: domain.PermLeadUpdate, Handler: wc.BriefSend},
		{Path: "/brief/confirm/:id", Perm: domain.PermLeadUpdate, Handler: wc.BriefConfirm},
		// 定制需求
		{Path: "/custom-request/list", Perm: domain.PermRequestView, Handler: wc.StaffCustomRequestList},
		{Path: "/custom-request/respond/:id", Perm: domain.PermRequestHandle, Handler: wc.CustomRequestRespond},
		// 评价
		{Path: "/review/list", Perm: domain.PermReviewView, Handler: wc.ReviewList},
		{Path: "/review/reply/:id", Perm: domain.PermReviewReply, Handler: wc.ReviewReply},
		// 客户档案：今日待跟进（跟进对象是线索）、手机号换绑
		{Path: "/customer/today-follow", Perm: domain.PermLeadView, Handler: wc.TodayFollow},
		{Path: "/customer/mobile", Perm: domain.PermCustomerUpdate, Handler: wc.CustomerMobileUpdate},
		// 交付反馈整理
		{Path: "/delivery/feedback/list", Perm: domain.PermDeliveryView, Handler: wc.FeedbackList},
		{Path: "/delivery/feedback/handle/:item_id", Perm: domain.PermDeliveryUpdate, Handler: wc.FeedbackHandle},
		// 个人中心：设备管理（操作对象是登录者本人设备，属自助类 → 免挂权限点）
		{Path: "/device/list", Handler: wc.DeviceList},
		{Path: "/device/remove/:id", Handler: wc.DeviceRemove},
	}
}
