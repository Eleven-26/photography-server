// 员工登录与手机号换绑（由 staff.go 按业务域拆分，2026-09-21 结构整改）。
package wechat

import (
	"photography-server/internal/contract"
	"photography-server/internal/middleware"
	"photography-server/internal/pkg/response"
	"photography-server/internal/presentation/bind"

	"github.com/gin-gonic/gin"
)

// StaffSmsCode 发送登录验证码（保留：员工端 UI 当前未调用，待手机验证码登录上线）
// @Summary      发送登录验证码
// @Description  员工登录短信验证码，场景固定为 login。**免鉴权**（登录发生在拿到令牌之前）。
// @Description  前端暂未调用：员工端主登录方式为「账号 + 密码」（/wechat/staff/auth/login）。
// @Tags         员工端·登录
// @Accept       json
// @Produce      json
// @Param        req  body  object{mobile=string}  true  "手机号"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Router       /wechat/staff/auth/sms-code [post]
func (h *Controller) StaffSmsCode(c *gin.Context) {
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

// StaffPasswordLogin 员工账号密码登录（与 PC 同一套凭据校验：bcrypt + 失败锁定）。
//
// 响应结构与 PC 的 /auth/login 完全一致：{ token, user: UserInfoVO }，
// user 内嵌 sys_user 全字段（含 id/username/nickname/avatar/mobile/role_id/store_id）
// 并追加 role_code / role_name / data_scope / permissions。
// @Summary      员工账号密码登录
// @Description  员工端主登录方式，与 PC 端 /auth/login 共用同一套凭据校验（bcrypt + 失败锁定）。**免鉴权**。
// @Description  响应结构与 PC 完全一致：{ token, user }，user 内嵌 sys_user 字段并追加 role_code / role_name / data_scope / permissions。
// @Tags         员工端·登录
// @Accept       json
// @Produce      json
// @Param        req  body  object{username=string,password=string,device_name=string,platform=string}  true  "登录凭据（platform: ios/android，选填）"
// @Success      200  {object}  response.Body{data=contract.LoginResp}
// @Failure      400  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Router       /wechat/staff/auth/login [post]
func (h *Controller) StaffPasswordLogin(c *gin.Context) {
	var req struct {
		Username   string `json:"username" binding:"required"` // 登录账号
		Password   string `json:"password" binding:"required"` // 登录密码
		DeviceName string `json:"device_name"`                 // 设备名称（选填，用于登录设备管理）
		Platform   string `json:"platform"`                    // ios/android
	}
	if err := bind.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	resp, err := h.Svc.StaffPasswordLogin(c.Request.Context(), req.Username, req.Password, req.DeviceName, req.Platform, c.ClientIP())
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, resp)
}

// StaffLoginByCode 员工手机号验证码登录（预留：员工端 UI 未接入，契约已就位）
// @Summary      员工验证码登录
// @Description  手机号 + 短信验证码登录。**免鉴权**；后端已就绪，员工端 UI 暂未接入（与 auth/sms-code 成对保留）。
// @Tags         员工端·登录
// @Accept       json
// @Produce      json
// @Param        req  body  object{mobile=string,code=string,device_name=string,platform=string}  true  "手机号与验证码（platform: ios/android，选填）"
// @Success      200  {object}  response.Body{data=contract.LoginResp}
// @Failure      400  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Router       /wechat/staff/auth/login-by-code [post]
func (h *Controller) StaffLoginByCode(c *gin.Context) {
	var req struct {
		Mobile     string `json:"mobile" binding:"required"`
		Code       string `json:"code" binding:"required"`
		DeviceName string `json:"device_name"` // 设备名称（选填，用于设备管理）
		Platform   string `json:"platform"`    // ios/android
	}
	if err := bind.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	resp, err := h.Svc.StaffSmsLogin(c.Request.Context(), req.Mobile, req.Code, req.DeviceName, req.Platform, c.ClientIP())
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, resp)
}

// StaffMobileCode 发送换绑验证码（发往**当前绑定手机号**，scene=change_mobile）
// @Summary      发送换绑验证码
// @Description  验证码发往**当前绑定的手机号**，场景 scene=change_mobile（与登录场景隔离）。免权限点。
// @Tags         员工端·个人中心
// @Accept       json
// @Produce      json
// @Success      200  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /wechat/staff/user/mobile-code [post]
func (h *Controller) StaffMobileCode(c *gin.Context) {
	op := middleware.GetOperator(c)
	if err := h.Svc.SendStaffMobileCode(c.Request.Context(), op); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

// StaffChangeMobile 校验验证码并换绑本人手机号（body: {code, new_mobile}）
// @Summary      换绑手机号
// @Description  校验验证码后换绑本人手机号。免权限点（操作对象是登录者本人账号）。
// @Tags         员工端·个人中心
// @Accept       json
// @Produce      json
// @Param        req  body  contract.StaffChangeMobileReq  true  "验证码与新手机号"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /wechat/staff/user/change-mobile [post]
func (h *Controller) StaffChangeMobile(c *gin.Context) {
	op := middleware.GetOperator(c)
	var req contract.StaffChangeMobileReq
	if err := bind.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.ChangeStaffMobile(c.Request.Context(), op, req.Code, req.NewMobile); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}
