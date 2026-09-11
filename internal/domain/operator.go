package domain

import "context"

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

// operatorCtxKey 操作人在 context 中的私有键类型（避免与其它包的同名键冲突）。
type operatorCtxKey struct{}

// WithOperator 把当前操作人写入 context。
//
// 认证中间件在通过认证后调用；controller 原样把 request context 透传给 service，
// service 再透传给 repository，使行级数据权限过滤**无需改动任何仓储方法签名**
// （repository 用 OperatorFrom 取，见 repository/scope.go 的 opOf）。
//
// 客户端（H5/小程序客户区）、定时任务与单测链路不写入该键，
// repository 取不到时按「无操作人」处理（不加过滤），与改造前行为完全一致。
func WithOperator(ctx context.Context, op Operator) context.Context {
	return context.WithValue(ctx, operatorCtxKey{}, op)
}

// OperatorFrom 从 context 取回操作人。
//
// ok=false 表示该链路未经员工认证（客户侧接口 / 定时任务 / 单测），
// 调用方应视为零值 Operator —— 零值 DataScope 在 repository.applyScope 中
// 走「不加条件」分支，因此这类链路不受数据权限影响。
func OperatorFrom(ctx context.Context) (Operator, bool) {
	op, ok := ctx.Value(operatorCtxKey{}).(Operator)
	return op, ok
}

// ClientUser 客户端（H5/小程序）登录上下文，由 CustomerAuth 中间件注入
// 重构 #32：从 service 包移到 domain 包，解决 middleware 依赖 service 的层次倒置问题
type ClientUser struct {
	CustomerID int64
	CompanyID  int64
	Mobile     string
	Name       string
}
