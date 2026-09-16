package routes

import (
	"photography-server/internal/presentation/h5"
)

// 客户区路由表（H5 与小程序客户区**共用同一份声明**）。
//
// 背景（2026-09-16）：客户区此前是唯一没进本包的路由——48 条以命令式 `g.POST` 内联在
// presentation/h5/h5.go 的 RegisterPublic / RegisterAuthed 里，与管理端的声明式表并存，
// 后果有二：
//  1. 路由清单与 52 个 handler 挤在同一个 1000 行文件里，读 handler 要先跳过注册段；
//  2. 客户区**完全不在护栏测试覆盖范围**——router/routes_test.go 只扫 Common 与 staffExtra，
//     意味着"同 Path 跨端必须同一 Handler"这条硬保证对客户区不成立。
//
// 现改为声明式：h5 与小程序客户区引用的是**同一份 []Route**（同一 Route 实例、同一 Handler
// 函数指针），两端口径由构造保证而非靠"记得一起改"。
//
// 鉴权口径：客户区**不使用 RBAC 权限点**。能否访问由挂载分组决定——
// ClientPublic 挂免鉴权组，ClientAuthed 挂 CustomerAuth 组（令牌 → 客户上下文）。
// 因此本文件所有 Route 的 Perm 一律留空，护栏测试（TestClientRouteTableIntegrity）
// 断言这一点，防止有人把员工权限点误挂到客户区。
//
// c 为 nil 时仅可用于枚举路径（对 nil 指针取方法值不会 panic，也不会真的调用），
// 供护栏测试静态提取路由集合而不必构造完整依赖。

// ClientPublic 客户区公开路由（免登录）。
//
// 租户由 slug 短链标识定位——query slug / X-Slug 头，服务端反查 company_id，
// 绝不接受客户端直传裸 company_id（#29 移除，防遍历枚举）。
func ClientPublic(c *h5.Controller) []Route {
	return []Route{
		// 登录（客户手机号验证码；未注册自动建档）
		{Path: "/auth/sms-code", Handler: c.SmsCode},
		{Path: "/auth/login", Handler: c.Login},
		// 浏览（预约主页）
		{Path: "/package/list", Handler: c.PackageList},
		{Path: "/package/detail/:id", Handler: c.PackageDetail},
		{Path: "/studio/info", Handler: c.StudioInfo},
		{Path: "/slot/list", Handler: c.SlotList},
		// 定制需求提交（游客/登录均可）
		{Path: "/custom-request/submit", Handler: c.CustomRequestSubmit},
		// 作品集（报告 H5）：预约主页展示，只出「已发布 + 公开」作品
		{Path: "/asset/list", Handler: c.AssetList},
		{Path: "/asset/detail/:id", Handler: c.AssetDetail},
	}
}

// ClientAuthed 客户区需登录路由（分组挂 CustomerAuth，令牌注入客户上下文）。
//
// 口径说明（2026-09-12 联调补齐）：
//   - 所有接口一律 **POST + JSON body**，路径参数用 :id/:order_id（后端不读 query，见 pkg/params）；
//   - `/delivery/detail/:id` 与 `/delivery/items/:id` 的 :id 均为 **order_id**（与 PC 端同语义，
//     按订单反查交付单）；真正的 delivery_id 出现在 `/delivery/select/:id` 等推进类接口上；
//   - 列表统一 `response.PageOK`（`{list,total,page,page_size}`），
//     逐单明细（退款/收款/改期）为不分页的业务集合，沿用 PC 同路径的裸数组口径。
func ClientAuthed(c *h5.Controller) []Route {
	return []Route{
		// 预约/订单
		{Path: "/order/submit", Handler: c.BookingSubmit},
		{Path: "/order/confirm/:id", Handler: c.BookingConfirm},
		{Path: "/order/cancel/:id", Handler: c.BookingCancel},
		{Path: "/order/list", Handler: c.OrderList},
		{Path: "/order/detail/:id", Handler: c.OrderDetail},
		// 拍前准备已读（biz_order.prep_read_at）
		{Path: "/order/prep/read/:id", Handler: c.OrderPrepRead},
		// 改期
		{Path: "/reschedule/apply/:order_id", Handler: c.RescheduleApply},
		{Path: "/reschedule/cancel/:id", Handler: c.RescheduleCancel},
		{Path: "/reschedule/list/:order_id", Handler: c.RescheduleList},
		// 退款
		{Path: "/refund/apply/:order_id", Handler: c.RefundApply},
		{Path: "/refund/list/:order_id", Handler: c.RefundList},
		{Path: "/refund/confirm/:id", Handler: c.RefundConfirm},
		// 收款：记录 / 登记转账（资金不经平台，仅登记） / 收款方式
		{Path: "/payment/list/:order_id", Handler: c.PaymentList},
		{Path: "/pay/mark", Handler: c.PaymentMark},
		{Path: "/payment-method/list", Handler: c.PaymentMethods},
		// 评价
		{Path: "/review/create/:order_id", Handler: c.ReviewCreate},
		// 我的评价（客户中心 → 我的评价；只读，按令牌内 customer_id 锁定归属）
		{Path: "/review/list", Handler: c.ReviewList},
		// 选片与交付
		{Path: "/delivery/detail/:id", Handler: c.DeliveryDetail},
		{Path: "/delivery/items/:id", Handler: c.DeliveryItems},
		{Path: "/delivery/select/:id", Handler: c.SelectPhotos},
		{Path: "/delivery/confirm-extra/:id", Handler: c.ConfirmExtra},
		{Path: "/delivery/confirm/:id", Handler: c.ConfirmDelivery},
		{Path: "/delivery/feedback/:item_id", Handler: c.FeedbackSubmit},
		// 定制需求（本人提交的历史需求）
		{Path: "/custom-request/list", Handler: c.CustomRequestList},
		// 客户中心（H5 CC01）：个人资料读写。
		// 字段白名单见 dto.ClientProfileUpdateReq —— crm_customer 与员工端共用一张表，
		// remark / tags / level / source / status 属工作室内部信息，不在客户端可改范围。
		{Path: "/customer/profile", Handler: c.CustomerProfile},
		{Path: "/customer/profile/update", Handler: c.CustomerProfileUpdate},
		// 定制需求页「选择门店 → 选择摄影师」的候选（仅客户历史服务过的门店/摄影师，
		// 外加本次分享链接的分享人；见 service.ClientPhotographerOptions）
		{Path: "/customer/photographer-options", Handler: c.PhotographerOptions},
		// 报价（报告 H1）：列表 / 详情 / 接受 / 提出修改
		{Path: "/quote/list", Handler: c.QuoteList},
		{Path: "/quote/detail/:id", Handler: c.QuoteDetail},
		{Path: "/quote/accept/:id", Handler: c.QuoteAccept},
		{Path: "/quote/modify/:id", Handler: c.QuoteModify},
		// 改期调度费（报告 H2）：详情含支付状态，支付走「上传凭证 → 工作室核验」
		{Path: "/reschedule/detail/:id", Handler: c.RescheduleDetail},
		{Path: "/reschedule/pay/:id", Handler: c.ReschedulePay},
		// 加片费试算（报告 H3）
		{Path: "/delivery/extra-quote/:id", Handler: c.ExtraQuote},
		// 拍摄需求修改（报告 H4）
		{Path: "/order/requirement/update/:id", Handler: c.OrderRequirementUpdate},
		// 站内通知（报告 H6）
		{Path: "/notification/list", Handler: c.NotificationList},
		{Path: "/notification/unread-count", Handler: c.NotificationUnreadCount},
		{Path: "/notification/read/:id", Handler: c.NotificationRead},
		{Path: "/notification/read-all", Handler: c.NotificationReadAll},
	}
}
