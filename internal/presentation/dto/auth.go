package dto

import "photography-server/internal/model"

// LoginReq 登录请求
type LoginReq struct {
	Username string `json:"username" binding:"required"` // 用户名
	Password string `json:"password" binding:"required"` // 密码
}

// LoginResp 登录响应
type LoginResp struct {
	Token string     `json:"token"` // JWT Token
	User  UserInfoVO `json:"user"`  // 用户信息（含角色与权限）
}

// UserInfoVO 登录 / 个人资料返回的用户信息：SysUser + 角色 + 权限 + 数据范围。
//
// 前端 auth store 依赖 role_code 与 permissions 做按钮级 / 路由级权限判定。
// 这两个字段此前从未下发（前端类型里声明了、实际恒为 undefined），本次补齐。
// 嵌入 model.SysUser 保证原有字段（id/username/nickname/avatar/role_id/company_id/store_id…）
// 的 JSON 结构完全不变，仅追加字段，对存量前端零破坏。
type UserInfoVO struct {
	model.SysUser
	RoleCode    string   `json:"role_code"`   // 角色编码 admin/manager/photographer/sales
	RoleName    string   `json:"role_name"`   // 角色名称
	DataScope   int      `json:"data_scope"`  // 数据范围 1-全部 2-本门店 3-仅本人
	Permissions []string `json:"permissions"` // 权限点集合（admin 角色为全量）
}

// ChangePasswordReq 修改密码
type ChangePasswordReq struct {
	OldPassword string `json:"old_password" binding:"required"` // 原密码
	NewPassword string `json:"new_password" binding:"required"` // 新密码
}
