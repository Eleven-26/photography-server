package dto

// ======================== 用户 ========================

// UserCreateReq 创建用户
type UserCreateReq struct {
	StoreID  int64  `json:"store_id"`                    // 门店ID
	Username string `json:"username" binding:"required"` // 登录账号
	Password string `json:"password" binding:"required"` // 密码
	Nickname string `json:"nickname"`                    // 昵称
	Mobile   string `json:"mobile"`                      // 手机号
	RoleID   int64  `json:"role_id" binding:"required"`  // 角色ID
	Status   int    `json:"status"`                      // 状态: 1启用 0禁用
}

// UserUpdateReq 更新用户
type UserUpdateReq struct {
	StoreID  int64  `json:"store_id"` // 门店ID
	Nickname string `json:"nickname"` // 昵称
	Mobile   string `json:"mobile"`   // 手机号
	Email    string `json:"email"`    // 邮箱
	Avatar   string `json:"avatar"`   // 头像URL
	RoleID   int64  `json:"role_id"`  // 角色ID
	Status   int    `json:"status"`   // 状态: 1启用 0禁用
}

// ======================== 角色 ========================

// RoleCreateReq 创建角色
type RoleCreateReq struct {
	Name   string `json:"name" binding:"required"` // 角色名称
	Code   string `json:"code" binding:"required"` // 角色编码
	Remark string `json:"remark"`                  // 备注
}

// RoleUpdateReq 更新角色
type RoleUpdateReq struct {
	Name   string `json:"name"`   // 角色名称
	Code   string `json:"code"`   // 角色编码
	Remark string `json:"remark"` // 备注
	Status int    `json:"status"` // 状态: 1启用 0禁用
}

// ======================== 门店 ========================

// StoreCreateReq 创建门店
type StoreCreateReq struct {
	Name    string `json:"name" binding:"required"` // 门店名称
	Address string `json:"address"`                 // 地址
	Phone   string `json:"phone"`                   // 联系电话
}

// StoreUpdateReq 更新门店
type StoreUpdateReq struct {
	Name    string `json:"name"`    // 门店名称
	Address string `json:"address"` // 地址
	Phone   string `json:"phone"`   // 联系电话
	Status  int    `json:"status"`  // 状态: 1启用 0禁用
}

// ======================== 密码 ========================

// ResetPasswordReq 重置密码
type ResetPasswordReq struct {
	Password string `json:"password" binding:"required"` // 新密码
}
