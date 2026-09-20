package controller

import (
	"github.com/gin-gonic/gin"

	"photography-server/internal/contract"
	"photography-server/internal/middleware"
	"photography-server/internal/pkg/response"
	"photography-server/internal/presentation/bind"
)

// Login 登录
// @Summary      账号密码登录
// @Description  员工账号密码登录（pc / miniapp 管理端共用），成功后返回 JWT 与用户信息（含角色、权限点、数据范围）；连续失败会触发账号锁定。
// @Tags         认证
// @Accept       json
// @Produce      json
// @Param        req  body  contract.LoginReq  true  "登录请求"
// @Success      200  {object}  response.Body{data=contract.LoginResp}
// @Failure      400  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Router       /auth/login [post]
func (h *Controller) Login(c *gin.Context) {
	var req contract.LoginReq
	if err := bind.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	resp, err := h.Svc.Login(c.Request.Context(), h.Cfg.JWT.Secret, h.Cfg.JWT.Issuer, h.Cfg.JWT.ExpireHours, req, c.ClientIP())
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, resp)
}

// Profile 当前登录用户信息
// @Summary      当前登录用户信息
// @Description  返回登录员工的资料、角色、权限点与数据范围，供前端做按钮级 / 路由级权限判定。
// @Tags         认证
// @Accept       json
// @Produce      json
// @Success      200  {object}  response.Body{data=contract.UserInfoVO}
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /user/profile [post]
func (h *Controller) Profile(c *gin.Context) {
	op := middleware.GetOperator(c)
	u, err := h.Svc.Profile(c.Request.Context(), op)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, u)
}

// ChangePassword 修改密码
// @Summary      修改密码
// @Description  校验原密码后修改当前登录员工密码。
// @Tags         认证
// @Accept       json
// @Produce      json
// @Param        req  body  contract.ChangePasswordReq  true  "修改密码请求"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /user/change-password [post]
func (h *Controller) ChangePassword(c *gin.Context) {
	op := middleware.GetOperator(c)
	var req contract.ChangePasswordReq
	if err := bind.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.ChangePassword(c.Request.Context(), op, req.OldPassword, req.NewPassword); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

// Logout 登出：jti 进黑名单使当前令牌立即失效（#15），替代旧"无状态 JWT 仅返回成功"
// @Summary      登出
// @Description  将当前令牌 jti 加入黑名单，使令牌立即失效（替代旧的无状态 JWT 仅返回成功）。
// @Tags         认证
// @Accept       json
// @Produce      json
// @Success      200  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /user/logout [post]
func (h *Controller) Logout(c *gin.Context) {
	token := bearerToken(c)
	if err := h.Svc.Logout(c.Request.Context(), token); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}
