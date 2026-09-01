package dto

type UserCreateReq struct {
	StoreID  int64  `json:"store_id"`
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Nickname string `json:"nickname"`
	Mobile   string `json:"mobile"`
	RoleID   int64  `json:"role_id" binding:"required"`
	Status   int    `json:"status"`
}

type UserUpdateReq struct {
	StoreID  int64  `json:"store_id"`
	Nickname string `json:"nickname"`
	Mobile   string `json:"mobile"`
	Email    string `json:"email"`
	Avatar   string `json:"avatar"`
	RoleID   int64  `json:"role_id"`
	Status   int    `json:"status"`
}

type RoleCreateReq struct {
	Name   string `json:"name" binding:"required"`
	Code   string `json:"code" binding:"required"`
	Remark string `json:"remark"`
}

type RoleUpdateReq struct {
	Name   string `json:"name"`
	Code   string `json:"code"`
	Remark string `json:"remark"`
	Status int    `json:"status"`
}

type StoreCreateReq struct {
	Name    string `json:"name" binding:"required"`
	Address string `json:"address"`
	Phone   string `json:"phone"`
}

type StoreUpdateReq struct {
	Name    string `json:"name"`
	Address string `json:"address"`
	Phone   string `json:"phone"`
	Status  int    `json:"status"`
}

type ResetPasswordReq struct {
	Password string `json:"password" binding:"required"`
}
