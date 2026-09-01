package dto

import "photography-server/internal/model"

// LoginReq 登录请求
type LoginReq struct {
	Username string `json:"username" binding:"required"` // 用户名
	Password string `json:"password" binding:"required"` // 密码
}

// LoginResp 登录响应
type LoginResp struct {
	Token string        `json:"token"` // JWT Token
	User  model.SysUser `json:"user"`  // 用户信息
}

// ChangePasswordReq 修改密码
type ChangePasswordReq struct {
	OldPassword string `json:"old_password" binding:"required"` // 原密码
	NewPassword string `json:"new_password" binding:"required"` // 新密码
}
