package domain

// Operator 当前操作人（由认证中间件注入）
// 重构 #32：从 service 包移到 domain 包，解决 middleware 依赖 service 的层次倒置问题
type Operator struct {
	UserID    int64
	Username  string
	Nickname  string
	CompanyID int64
	StoreID   int64
	RoleID    int64

	// 以下字段由认证阶段加载角色授权信息后填充（见 middleware/auth.go）。
	// RoleCode 用于 admin 短路判定；DataScope 供 repository 层行级过滤；
	// Permissions 为权限点集合（元素形如 "order:view"）。
	// 权限体系未启用（角色无任何配置）时 DataScope 为零值、Permissions 为空切片。
	RoleCode    RoleCode
	DataScope   DataScope
	Permissions []string
}

// ClientUser 客户端（H5/小程序）登录上下文，由 CustomerAuth 中间件注入
// 重构 #32：从 service 包移到 domain 包，解决 middleware 依赖 service 的层次倒置问题
type ClientUser struct {
	CustomerID int64
	CompanyID  int64
	Mobile     string
	Name       string
}
