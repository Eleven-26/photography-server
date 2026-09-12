package controller

import (
	"github.com/gin-gonic/gin"

	"photography-server/internal/app"
	"photography-server/internal/config"
	"photography-server/internal/service"
)

// Controller 所有接口处理器统一挂在 Controller 上
type Controller struct {
	Svc *service.Service
	Cfg *config.Config
	// App 依赖容器，仅供非 release 环境注册的调试路由（/test/*）使用（#40 由组合根注入）。
	App *app.App
}

func New(svc *service.Service, cfg *config.Config, a *app.App) *Controller {
	return &Controller{Svc: svc, Cfg: cfg, App: a}
}

// bearerToken 从 Authorization: Bearer 头取令牌（登出/吊销等场景用）
func bearerToken(c *gin.Context) string {
	auth := c.GetHeader("Authorization")
	if len(auth) > 7 && auth[:7] == "Bearer " {
		return auth[7:]
	}
	return ""
}

// 说明（2026-09-12 路由整理）：原先本文件自带的 bindJSON / pager / queryStr / queryInt /
// pathID / pathParam 六个 helper 已收敛到 internal/presentation/bind，
// 与 h5 / wechat 端共用同一实现（此前三处各一份、签名还不一致）。
