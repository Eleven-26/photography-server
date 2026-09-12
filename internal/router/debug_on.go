//go:build debug

package router

import (
	"github.com/gin-gonic/gin"

	"photography-server/internal/presentation/controller"
)

// registerDebugRoutes 注册调试路由（带 -tags debug 构建时编译本文件）。
//
// 两级把关：
//  1. 编译期——本文件与 controller/test.go 均带 debug 构建标签，默认构建（含生产镜像）
//     不含这些代码，调试接口在二进制层面即不存在；
//  2. 运行期——profile 白名单（dev/test/docker.dev），prod 即使误配也不注册。
func registerDebugRoutes(g *gin.RouterGroup, ctl *controller.Controller, profile string) {
	if !debugProfile(profile) {
		return
	}
	// 配置密文生成：运维刚需，与构建标签无关，两个分支都注册
	registerDebugTools(g, ctl)
	// 基础设施调试接口：仅 debug 构建存在
	registerDebugInfra(g, ctl)
}

// registerDebugInfra 注册基础设施调试路由（Redis / NATS / ES / Mongo / Jaeger 连通性与读写实验）。
//
// 处理器见 presentation/controller/test.go（同样带 debug 标签），通过 App 容器注入的句柄访问
// Redis/NATS/ES/Mongo，不经过 service 层，也未挂业务鉴权——禁止在生产环境启用。
func registerDebugInfra(g *gin.RouterGroup, ctl *controller.Controller) {
	t := g.Group("/test")
	t.POST("/redis/ping", ctl.RedisPing)
	t.POST("/redis/set", ctl.RedisSet)
	t.POST("/redis/get", ctl.RedisGet)
	t.POST("/redis/del", ctl.RedisDel)
	t.POST("/nats/status", ctl.NATSStatus)
	t.POST("/nats/pub", ctl.NATSPub)
	t.POST("/nats/pub-persistent", ctl.NATSPubPersistent)
	t.POST("/nats/pub-pull", ctl.NATSPubPull)
	t.POST("/nats/request", ctl.NATSRequest)
	t.POST("/es/status", ctl.ESStatus)
	t.POST("/es/index", ctl.ESIndex)
	t.POST("/es/search", ctl.ESSearch)
	t.POST("/es/list", ctl.ESList)
	t.POST("/es/delete", ctl.ESDelete)
	t.POST("/mongo/status", ctl.MongoStatus)
	t.POST("/mongo/insert", ctl.MongoInsert)
	t.POST("/mongo/insert-many", ctl.MongoInsertMany)
	t.POST("/mongo/find", ctl.MongoFind)
	t.POST("/mongo/find-one", ctl.MongoFindOne)
	t.POST("/mongo/update", ctl.MongoUpdate)
	t.POST("/mongo/delete", ctl.MongoDelete)
	t.POST("/mongo/delete-by-id", ctl.MongoDeleteByID)
	t.POST("/jaeger/status", ctl.JaegerStatus)
	t.POST("/jaeger/trace", ctl.JaegerTrace)

	t.POST("/test", ctl.Test)
}
