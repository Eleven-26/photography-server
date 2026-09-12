package routes

import (
	"photography-server/internal/domain"
	"photography-server/internal/presentation/controller"
)

// calendarFinance 档期 / 财务。
//
// 档期规则（排班时段模板）挂在 /calendar 组下。
// 财务导出单独收敛：可见不等于可带走（finance:export 独立于 finance:view）。
func calendarFinance(ctl *controller.Controller) []Route {
	return []Route{
		{Path: "/calendar/list", Perm: domain.PermCalendarView, Handler: ctl.CalendarList},
		{Path: "/calendar/lock", Perm: domain.PermCalendarUpdate, Handler: ctl.CalendarLock},
		{Path: "/calendar/cancel/:id", Perm: domain.PermCalendarUpdate, Handler: ctl.CalendarCancel},
		{Path: "/calendar/slot-template/list", Perm: domain.PermCalendarView, Handler: ctl.SlotTemplateList},
		{Path: "/calendar/slot-template/save", Perm: domain.PermCalendarUpdate, Handler: ctl.SlotTemplateSave},
		{Path: "/calendar/slot-template/save/:id", Perm: domain.PermCalendarUpdate, Handler: ctl.SlotTemplateSave},
		{Path: "/calendar/slot-template/delete/:id", Perm: domain.PermCalendarUpdate, Handler: ctl.SlotTemplateDelete},

		{Path: "/finance/summary", Perm: domain.PermFinanceView, Handler: ctl.FinanceSummary},
		{Path: "/finance/payments", Perm: domain.PermFinanceView, Handler: ctl.FinancePayments},
		{Path: "/finance/refunds", Perm: domain.PermFinanceView, Handler: ctl.FinanceRefunds},
		{Path: "/finance/export", Perm: domain.PermFinanceExport, Handler: ctl.FinanceExport},
	}
}
