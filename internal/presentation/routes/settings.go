package routes

import (
	"photography-server/internal/domain"
	"photography-server/internal/presentation/controller"
)

// settingsMisc 工作台 / 通知 / 设置。
//
// 通知：读通知是每个员工的基础能力，notification:view 控制"能否进入通知中心"。
// 设置：operation-log 归 log:view；收款方式维护归 settings:update。
func settingsMisc(ctl *controller.Controller) []Route {
	return []Route{
		{Path: "/dashboard/overview", Perm: domain.PermDashboardView, Handler: ctl.DashboardOverview},

		{Path: "/notification/list", Perm: domain.PermNotificationView, Handler: ctl.NotificationList},
		{Path: "/notification/unread-count", Perm: domain.PermNotificationView, Handler: ctl.NotificationUnreadCount},
		{Path: "/notification/read/:id", Perm: domain.PermNotificationView, Handler: ctl.NotificationRead},
		{Path: "/notification/read-all", Perm: domain.PermNotificationView, Handler: ctl.NotificationReadAll},

		{Path: "/settings/workspace", Perm: domain.PermSettingsView, Handler: ctl.Workspace},
		{Path: "/settings/company/update", Perm: domain.PermSettingsUpdate, Handler: ctl.CompanyUpdate},
		{Path: "/settings/payment-method/list", Perm: domain.PermSettingsView, Handler: ctl.PaymentMethodList},
		{Path: "/settings/payment-method/create", Perm: domain.PermSettingsUpdate, Handler: ctl.PaymentMethodCreate},
		{Path: "/settings/payment-method/update/:id", Perm: domain.PermSettingsUpdate, Handler: ctl.PaymentMethodUpdate},
		{Path: "/settings/payment-method/delete/:id", Perm: domain.PermSettingsUpdate, Handler: ctl.PaymentMethodDelete},
		{Path: "/settings/operation-log/list", Perm: domain.PermLogView, Handler: ctl.OperationLogList},
		{Path: "/settings/studio/get", Perm: domain.PermSettingsView, Handler: ctl.StudioGet},
		{Path: "/settings/studio/update", Perm: domain.PermSettingsUpdate, Handler: ctl.StudioUpdate},
	}
}
