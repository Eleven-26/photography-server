// 登录与短信验证码（由 h5.go 按业务域拆分，2026-09-21 结构整改）。
package h5

import (
	"photography-server/internal/pkg/errs"
	"photography-server/internal/pkg/response"
	"photography-server/internal/presentation/bind"

	"github.com/gin-gonic/gin"
)

// SmsCode 发送登录验证码
// @Summary      发送登录验证码
// @Description  短信发往指定手机号，场景固定为 login；短信通道未接入时验证码只打服务端日志。
// @Tags         客户区·登录
// @Accept       json
// @Produce      json
// @Param        req  body  object{mobile=string}  true  "手机号"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Router       /h5/auth/sms-code [post]
func (h *Controller) SmsCode(c *gin.Context) {
	var req struct {
		Mobile string `json:"mobile" binding:"required"`
	}
	if err := bind.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.SendSmsCode(c.Request.Context(), "login", req.Mobile); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

// Login 客户手机号验证码登录（未注册自动建档），openid 为小程序场景透传。
// 租户定位：body slug（可选）→ query slug / X-Slug 头 → 服务端反查 company_id（#29）
// @Summary      客户登录
// @Description  手机号 + 短信验证码登录；**未注册的手机号自动建档客户**（手机号为租户内唯一键）。
// @Description  租户定位：body.slug（也可放 query slug / X-Slug 头）；服务端按 slug 反查 company_id，**不接受客户端直传 company_id**。
// @Description  校验规则：dev / docker.dev 允许免验证码登录（短信通道未接入），test / prod 及未知 profile 强制校验验证码。
// @Description  登录成功返回客户令牌，后续请求以 "Bearer <token>" 放入 Authorization 头。
// @Tags         客户区·登录
// @Accept       json
// @Produce      json
// @Param        req  body  object{slug=string,mobile=string,code=string,openid=string}  true  "登录信息（mobile 必填；openid 为小程序场景透传）"
// @Success      200  {object}  response.Body{data=object{token=string,customer=model.Customer}}
// @Failure      400  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Router       /h5/auth/login [post]
func (h *Controller) Login(c *gin.Context) {
	var req struct {
		Slug   string `json:"slug"` // 预约主页短链标识（也可放 query/头）
		Mobile string `json:"mobile" binding:"required"`
		Code   string `json:"code"` // 短信验证码；开发环境免验证码时可为空（见 loginRequireSmsCode）
		OpenID string `json:"openid"`
	}
	if err := bind.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	slug := req.Slug
	if slug == "" {
		slug = slugFrom(c)
	}
	if slug == "" {
		response.Fail(c, errs.BadRequest(errs.ErrSlugRequired))
		return
	}
	companyID, err := h.Svc.ResolveCompanyBySlug(c.Request.Context(), slug)
	if err != nil {
		response.Fail(c, errs.Internal(""))
		return
	}
	if companyID <= 0 {
		response.Fail(c, errs.BadRequest(errs.ErrHomepageNotConfigured))
		return
	}
	customer, token, err := h.Svc.CustomerSmsLogin(c.Request.Context(), companyID, req.Mobile, req.Code, req.OpenID, h.loginRequireSmsCode())
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{
		"token":    token,
		"customer": customer,
	})
}

// loginRequireSmsCode 登录是否必须短信验证码。
//
// 仅本地开发环境（dev / docker.dev）放行「免验证码登录」：短信通道尚未接入，
// 验证码只打到服务端日志，联调时逐个去日志捞不现实。
//
// 为什么不做成配置项：免验证码 == 「知道手机号即可登录该客户账号」，而客户账号能读
// 自己的订单、交付样片、评价与个人资料。这种开关一旦可配，就有被误配到生产的风险
// （且误配后没有任何报错，直到有人发现能拿别人手机号登录），故写成**白名单判断**：
// 只有显式跑在 dev / docker.dev 才放开，test / prod 及一切未知 profile 一律强制校验。
// 若将来确有其它环境需要，改这个 switch —— 不要在配置里开一个自由开关。
func (h *Controller) loginRequireSmsCode() bool {
	switch h.Cfg.App.Profile {
	case "dev", "docker.dev":
		return false
	}
	return true
}
