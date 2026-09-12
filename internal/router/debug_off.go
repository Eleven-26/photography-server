//go:build !debug

package router

import (
	"github.com/gin-gonic/gin"

	"photography-server/internal/presentation/controller"
)

// registerDebugRoutes 是默认构建（无 debug 标签）下的实现：不注册基础设施调试路由。
//
// Redis / NATS / ES / Mongo / Jaeger 的调试处理器及其路由代码带 debug 构建标签，
// 本构建中根本不存在（不是"注册了但不生效"）；需要时以 -tags debug 构建
// （本地 Makefile 默认携带，容器内在 .env 设 GO_BUILD_TAGS=debug）。
//
// 仍保留 registerDebugTools（配置密文生成）——它是运维刚需且生产也需要，
// 由 profile 白名单把关，不随构建标签摘除。
func registerDebugRoutes(g *gin.RouterGroup, ctl *controller.Controller, profile string) {
	if !debugProfile(profile) {
		return
	}
	registerDebugTools(g, ctl)
}
