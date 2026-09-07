package wechat

import (
	"github.com/gin-gonic/gin"

	"photography-server/internal/config"
	h5ctl "photography-server/internal/presentation/h5"
	"photography-server/internal/service"
)

// Controller 微信小程序客户端接口。
// 小程序与 H5 的客户能力一致（登录/预约/订单/选片/评价），业务逻辑完全复用 h5 包；
// 本包只做入口隔离：独立路由前缀 /wechat，便于按端统计、鉴权与灰度。
// 小程序专属能力（openid 静默登录、微信支付）在 P1 扩展，届时在本包增量实现。
type Controller struct {
	*h5ctl.Controller
}

func New(svc *service.Service, cfg *config.Config) *Controller {
	return &Controller{Controller: h5ctl.New(svc, cfg)}
}

// RegisterPublic 注册公开路由（无需登录）
func (h *Controller) RegisterPublic(g *gin.RouterGroup) {
	h.Controller.RegisterPublic(g)
}

// RegisterAuthed 注册需登录路由（CustomerAuth 注入 ClientUser）
func (h *Controller) RegisterAuthed(g *gin.RouterGroup) {
	h.Controller.RegisterAuthed(g)
}
