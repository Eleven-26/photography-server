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
}

// ClientUser 客户端（H5/小程序）登录上下文，由 CustomerAuth 中间件注入
// 重构 #32：从 service 包移到 domain 包，解决 middleware 依赖 service 的层次倒置问题
type ClientUser struct {
	CustomerID int64
	CompanyID  int64
	Mobile     string
	Name       string
}
