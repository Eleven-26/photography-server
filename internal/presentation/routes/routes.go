// Package routes 定义端无关的业务路由表：把「业务路由内容」与「端暴露面」拆开。
//
// 背景（2026-09-12 路由整理）：原先管理端路由是 router.go 里一个 170 行的私有函数
// registerCommon，且硬绑 *controller.Controller——wechat 包引用不到，于是员工端只能把
// 24 条同路径路由连同 handler 一起抄了一遍（48 条里 34 条、71% 是重复注册，且已产生契约漂移：
// 同路径 /order/list 在两端返回结构不同）。根因不是约定有问题，而是约定没被跨包复用。
//
// 现把业务路由抽成端无关的数据表，各端只声明「暴露哪些 Path」（见 router/endpoints.go）。
// 同一 Path 在任意两端引用的是同一个 Route 实例，Handler 是同一个函数指针——
// 护栏测试（router/routes_test.go）据此断言，杜绝再次复制。
package routes

import (
	"github.com/gin-gonic/gin"

	"photography-server/internal/domain"
	"photography-server/internal/presentation/controller"
)

// Route 一条端无关的业务路由。
type Route struct {
	// Path 端内相对路径（含分组前缀，如 "/order/list"）。
	// 同一业务在所有端使用同一 Path；端有历史包袱的异路径在端声明里另行登记（见 StaffAliases）。
	Path string
	// Perm 权限点；空值表示免挂（自助类 / 通用能力），必须在豁免白名单内（见 router 包断言测试）。
	Perm domain.Perm
	// Handler 处理器。跨端复用时必须是同一函数指针——护栏测试据此杜绝重复实现。
	Handler gin.HandlerFunc
}

// Common 返回管理端业务路由的唯一实现：PC 与小程序全量挂载，员工端复用其子集。
//
// ctl 为 nil 时仅可用于枚举路径（对 nil 指针取方法值不会 panic，也不会真的调用），
// 供护栏测试静态提取路由集合而不必构造完整依赖。
func Common(ctl *controller.Controller) []Route {
	var all []Route
	all = append(all, userRoleStore(ctl)...)
	all = append(all, customerLeadQuote(ctl)...)
	all = append(all, catalog(ctl)...)
	all = append(all, orderFlow(ctl)...)
	all = append(all, paymentRefundDelivery(ctl)...)
	all = append(all, calendarFinance(ctl)...)
	all = append(all, settingsMisc(ctl)...)
	return all
}

// Paths 返回全部业务路由的路径（运维排查与测试用）。
func Paths(ctl *controller.Controller) []string {
	all := Common(ctl)
	out := make([]string, 0, len(all))
	for _, r := range all {
		out = append(out, r.Path)
	}
	return out
}
