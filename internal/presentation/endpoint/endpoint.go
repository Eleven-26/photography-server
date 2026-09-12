// Package endpoint 提供端暴露面的挂载机制：把端无关路由表（routes.Route）
// 按端声明（Endpoint）挂到 gin 路由组上。
//
// 分工：routes 包是「数据」（有哪些业务路由），endpoint 包是「机制」（怎么挂），
// router/endpoints.go 是「声明」（每个端暴露哪些）。三者解耦后，员工端不再需要复制 handler：
// 同 Path 的路由引用同一个 routes.Route 实例，Handler 是同一个函数指针。
package endpoint

import (
	"github.com/gin-gonic/gin"

	"photography-server/internal/middleware"
	"photography-server/internal/presentation/routes"
)

// Endpoint 一个端的暴露面声明。
type Endpoint struct {
	// Name 端名（pc / miniapp / staff / ...），用于日志与断言失败信息。
	Name string
	// Prefix 挂载前缀：""（管理端）| "/miniapp" | "/wechat/staff"。
	Prefix string
	// Middlewares 分组级中间件（认证 / 操作日志），按顺序挂载。
	Middlewares []gin.HandlerFunc
	// Include 从公共路由表复用的 Path 白名单；nil = 全量（管理端语义）。
	// 白名单内的 Path 必须存在于公共路由表——同一 Path 在所有端引用同一 routes.Route
	// 实例（同一 Handler 函数指针），护栏测试据此断言。
	Include []string
	// Extra 端独有路由：异路径别名（Handler 仍复用公共实现）与真实端差异实现。
	Extra []routes.Route
}

// Mount 在 parent 下新建分组并挂载端点声明的全部路由。
// common 为端无关路由表（routes.Common 的返回值）。
func Mount(parent *gin.RouterGroup, ep Endpoint, common []routes.Route, mw *middleware.Middlewares) {
	g := parent.Group(ep.Prefix, ep.Middlewares...)
	for _, r := range common {
		if !ep.includes(r.Path) {
			continue
		}
		post(g, r, mw)
	}
	for _, r := range ep.Extra {
		post(g, r, mw)
	}
}

// includes 判断某路径是否被本端暴露：Include 为 nil 视为全量。
func (ep Endpoint) includes(path string) bool {
	if ep.Include == nil {
		return true
	}
	for _, p := range ep.Include {
		if p == path {
			return true
		}
	}
	return false
}

// post 注册一条路由：权限点非空时先挂 mw.Perm 判定（空值 = 自助类/通用能力，豁免）。
func post(g *gin.RouterGroup, r routes.Route, mw *middleware.Middlewares) {
	if r.Perm != "" {
		g.POST(r.Path, mw.Perm(r.Perm), r.Handler)
		return
	}
	g.POST(r.Path, r.Handler)
}
