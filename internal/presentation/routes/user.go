package routes

import (
	"photography-server/internal/domain"
	"photography-server/internal/presentation/controller"
)

// userRoleStore 用户 / 角色 / 门店 / 通用上传。
//
// 【免挂权限点的路由（自助类 / 通用能力）】——有意为之，不是遗漏：
//   - /user/profile、/user/change-password、/user/logout：操作对象是登录者本人账号，
//     不属于角色能力边界，任何登录员工都必须可用（否则改密码都要管理员授权，属设计缺陷）。
//   - /upload/file：通用文件上传能力，被收款凭证、作品、交付文件等多条链路共用；
//     挂 asset:upload 会连带拦掉销售上传收款凭证（sales 无 asset:upload），属误伤。
//     上传内容的安全性由各业务接口自身的归属校验与文件类型白名单保证。
//
// 除此之外每条业务路由都必须挂权限点。
func userRoleStore(ctl *controller.Controller) []Route {
	return []Route{
		// 用户与权限
		{Path: "/user/profile", Handler: ctl.Profile},
		{Path: "/user/change-password", Handler: ctl.ChangePassword},
		{Path: "/user/logout", Handler: ctl.Logout},
		{Path: "/user/list", Perm: domain.PermUserView, Handler: ctl.UserList},
		{Path: "/user/create", Perm: domain.PermUserCreate, Handler: ctl.UserCreate},
		{Path: "/user/update/:id", Perm: domain.PermUserUpdate, Handler: ctl.UserUpdate},
		{Path: "/user/delete/:id", Perm: domain.PermUserDelete, Handler: ctl.UserDelete},
		{Path: "/user/reset-password/:id", Perm: domain.PermUserResetPwd, Handler: ctl.UserResetPassword},

		// 角色（catalog/permissions 为配置入口的只读视图，归 role:view；
		// 二者随接口一同挂载权限点，避免 B1-1 上线到 B1-2 挂载之间出现可被任意登录员工改权限的安全空窗）
		{Path: "/role/list", Perm: domain.PermRoleView, Handler: ctl.RoleList},
		{Path: "/role/create", Perm: domain.PermRoleCreate, Handler: ctl.RoleCreate},
		{Path: "/role/update/:id", Perm: domain.PermRoleUpdate, Handler: ctl.RoleUpdate},
		{Path: "/role/delete/:id", Perm: domain.PermRoleDelete, Handler: ctl.RoleDelete},
		{Path: "/role/catalog", Perm: domain.PermRoleView, Handler: ctl.RoleCatalog},
		{Path: "/role/permissions/:id", Perm: domain.PermRoleView, Handler: ctl.RolePerms},
		{Path: "/role/grant/:id", Perm: domain.PermRoleGrant, Handler: ctl.RoleGrant},

		// 门店
		{Path: "/store/list", Perm: domain.PermStoreView, Handler: ctl.StoreList},
		{Path: "/store/create", Perm: domain.PermStoreCreate, Handler: ctl.StoreCreate},
		{Path: "/store/update/:id", Perm: domain.PermStoreUpdate, Handler: ctl.StoreUpdate},
		{Path: "/store/delete/:id", Perm: domain.PermStoreDelete, Handler: ctl.StoreDelete},

		// 通用上传
		{Path: "/upload/file", Handler: ctl.UploadFile},
	}
}
