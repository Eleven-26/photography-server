package wechat

import (
	"photography-server/internal/config"
	h5ctl "photography-server/internal/presentation/h5"
	"photography-server/internal/service"
)

// Controller 微信小程序端接口（移动端统一入口，当前无独立 App）。
// 小程序承载两类角色，路由按子前缀区分：
//   - 客户区 /wechat/...        ：与 H5 的客户能力一致（登录/预约/订单/选片/评价）。
//   - 员工区 /wechat/staff/...  ：摄影师/助理用小程序处理订单、日程、线索 AI 简报等。
//
// 路由注册（2026-09-16）：**客户区与员工区路由都不在本包**，本包只留 handler 实现。
//
//   - 客户区：声明在 presentation/routes/client.go（ClientPublic / ClientAuthed），
//     与 H5 **共用同一份 []Route**——同一 Route 实例、同一 Handler 函数指针，
//     仅挂载前缀不同（/h5 与 /wechat，见 router/router.go）。两端口径由构造保证，
//     不再依赖"记得两边一起改"。此前是靠本包转调 h5 的注册函数实现复用，
//     但那意味着客户区路由游离在声明式表之外、不受跨端护栏覆盖。
//   - 员工区：声明在 router/endpoints.go 的 staffEndpoint
//     （Include 复用管理端路由表 + Extra 端独有 + PublicExtra 登录出口）。
//
// 嵌入 *h5.Controller 保留的原因：共享 Svc/Cfg，并为将来小程序专属客户能力
// （openid 静默登录、微信支付）保留直接复用 H5 handler 的通道——届时增量实现即可，
// 不必把客户区逻辑再写一遍。
type Controller struct {
	*h5ctl.Controller
}

func New(svc *service.Service, cfg *config.Config) *Controller {
	return &Controller{Controller: h5ctl.New(svc, cfg)}
}
