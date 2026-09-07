package wechat

import (
	"github.com/gin-gonic/gin"

	"photography-server/internal/config"
	h5ctl "photography-server/internal/presentation/h5"
	"photography-server/internal/service"
)

// Controller 微信小程序端接口（移动端统一入口，当前无独立 App）。
// 小程序承载两类角色，路由按子前缀区分：
//   - 客户区 /wechat/...        ：与 H5 的客户能力一致（登录/预约/订单/选片/评价），
//     业务逻辑完全复用 h5 包，本包只做入口隔离（独立路由前缀，便于按端统计与灰度）。
//   - 员工区 /wechat/staff/...  ：摄影师/助理用小程序处理订单、日程、线索 AI 简报等，
//     见 staff.go（StaffAuth 员工认证）。
//
// 小程序专属能力（openid 静默登录、微信支付）在 P1 扩展，届时在本包增量实现。
type Controller struct {
	*h5ctl.Controller
}

func New(svc *service.Service, cfg *config.Config) *Controller {
	return &Controller{Controller: h5ctl.New(svc, cfg)}
}

// RegisterPublic 注册客户区公开路由（无需登录，挂 /wechat）
func (h *Controller) RegisterPublic(g *gin.RouterGroup) {
	h.Controller.RegisterPublic(g)
}

// RegisterAuthed 注册客户区需登录路由（CustomerAuth 注入 ClientUser，挂 /wechat）
func (h *Controller) RegisterAuthed(g *gin.RouterGroup) {
	h.Controller.RegisterAuthed(g)
}
