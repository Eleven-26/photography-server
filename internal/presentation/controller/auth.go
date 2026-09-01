package controller

import (
	"github.com/gin-gonic/gin"

	"photography-server/internal/middleware"
	"photography-server/internal/presentation/dto"
	"photography-server/internal/response"
)

// Login 登录
func (h *Controller) Login(c *gin.Context) {
	var req dto.LoginReq
	if err := h.bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	resp, err := h.Svc.Login(h.Cfg.JWT.Secret, h.Cfg.JWT.Issuer, h.Cfg.JWT.ExpireHours, req, c.ClientIP())
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, resp)
}

// Profile 当前登录用户信息
func (h *Controller) Profile(c *gin.Context) {
	op := middleware.GetOperator(c)
	u, err := h.Svc.Profile(op)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, u)
}

// ChangePassword 修改密码
func (h *Controller) ChangePassword(c *gin.Context) {
	op := middleware.GetOperator(c)
	var req dto.ChangePasswordReq
	if err := h.bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.ChangePassword(op, req.OldPassword, req.NewPassword); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

// Logout 登出（无状态 JWT，仅返回成功）
func (h *Controller) Logout(c *gin.Context) {
	response.OKNil(c)
}
