package router

import (
	"github.com/gin-gonic/gin"

	"photography-server/internal/presentation/controller"
)

// debugProfile 判断当前 profile 是否允许注册调试路由。
//
// 以 profile（服务端启动参数 / APP_PROFILE，可信）白名单为准——只有 dev/test/docker.dev 注册；
// 生产（prod）无论 app.mode 是否误配为 debug/空 都不暴露（旧实现按 mode != "release" 黑名单判断，
// mode 为空即等于非 release → 生产误配会全量暴露，是 P0 隐患）。
func debugProfile(profile string) bool {
	switch profile {
	case "dev", "test", "docker.dev":
		return true
	}
	return false
}

// registerDebugTools 注册**不受 debug 构建标签隔离**的调试工具路由。
//
// 目前仅 /test/config/encrypt（配置密文生成）：只加密、不解密，是新增敏感配置字段时的运维刚需
// （生产同样需要；另有 CLI 兜底 `go run ./cmd/configctl encrypt -v '明文'`），
// 因此不随基础设施调试接口一起在编译期摘除，只由 profile 白名单把关。
func registerDebugTools(g *gin.RouterGroup, ctl *controller.Controller) {
	t := g.Group("/test")
	t.POST("/config/encrypt", ctl.ConfigEncrypt)
}
