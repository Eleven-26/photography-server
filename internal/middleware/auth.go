package middleware

import (
	"context"

	"github.com/gin-gonic/gin"

	"photography-server/internal/domain"
	"photography-server/internal/model"
	"photography-server/internal/pkg/authcache"
	"photography-server/internal/pkg/errs"
	"photography-server/internal/pkg/jwtpkg"
	"photography-server/internal/presentation/response"
)

// extractToken 从 Authorization: Bearer 中提取令牌
func extractToken(c *gin.Context) string {
	auth := c.GetHeader("Authorization")
	if len(auth) > 7 && auth[:7] == "Bearer " {
		return auth[7:]
	}
	return ""
}

// tokenRevoked 检查 jti 是否已被吊销（登出黑名单）。
// Redis 不可用（m.Redis==nil）或查询出错时 fail-open 放行——吊销依赖 Redis 是尽力而为；
// 登录链路本身强依赖 Redis，Redis 长期不可用时服务已按 fail-fast 拒启（见 main.go）。
func (m *Middlewares) tokenRevoked(ctx context.Context, jti string) bool {
	if jti == "" {
		return false // 旧令牌无 jti，无吊销能力
	}
	rdb := m.Redis
	if rdb == nil {
		return false
	}
	n, err := rdb.Exists(ctx, jwtpkg.BlacklistKey(jti)).Result()
	return err == nil && n > 0
}

// loadStaffProfile 加载员工认证画像：优先 Redis 缓存（60s TTL），未命中回源 DB。
// 缓存命中但画像不全时同样回源 DB 覆盖（fail-open，保证租户/角色字段正确）。
func (m *Middlewares) loadStaffProfile(ctx context.Context, userID int64) (*authcache.StaffProfile, error) {
	rdb := m.Redis
	if rdb != nil {
		if p, hit, err := authcache.GetStaff(ctx, rdb, userID); err == nil && hit {
			return p, nil
		}
	}
	var u model.SysUser
	if err := m.DB.WithContext(ctx).First(&u, userID).Error; err != nil {
		return nil, err
	}
	p := &authcache.StaffProfile{
		Status:    u.Status,
		Username:  u.Username,
		Nickname:  u.Nickname,
		CompanyID: u.CompanyID,
		StoreID:   u.StoreID,
		RoleID:    u.RoleID,
	}
	if rdb != nil {
		authcache.SetStaff(ctx, rdb, userID, p)
	}
	return p, nil
}

// loadRoleAuth 加载角色授权信息（角色编码 + 数据范围 + 权限点集合）。
//
// 缓存优先（30min TTL）→ 未命中回源 DB → 回写缓存；Redis 不可用时直查 DB
// （fail-open：缓存故障不能导致全员 403）。
//
// 失败降级为「空权限」而非放行：角色不存在/已删除时返回空集合，该用户将无法通过
// 任何权限点判定（安全优先）。admin 角色不受影响（按角色码短路，见 hasPerm）。
func (m *Middlewares) loadRoleAuth(ctx context.Context, companyID, roleID int64) *authcache.RoleAuth {
	empty := &authcache.RoleAuth{Permissions: []string{}}
	if roleID <= 0 || m.DB == nil {
		return empty
	}
	if rdb := m.Redis; rdb != nil {
		if a, hit, err := authcache.GetRoleAuth(ctx, rdb, companyID, roleID); err == nil && hit && a != nil {
			return a
		}
	}
	db := m.DB.WithContext(ctx)
	var role model.SysRole
	if err := db.Where("company_id = ? AND id = ?", companyID, roleID).First(&role).Error; err != nil {
		return empty
	}
	a := &authcache.RoleAuth{
		RoleCode:    role.Code,
		DataScope:   role.DataScope,
		Permissions: make([]string, 0, 16),
	}
	var perms []string
	db.Model(&model.SysRolePermission{}).
		Where("company_id = ? AND role_id = ?", companyID, roleID).
		Order("permission ASC").
		Pluck("permission", &perms)
	// 过滤代码中已删除的权限点（常量被移除后表中遗留的行），避免脏数据进入判定
	for _, p := range perms {
		if domain.IsValidPerm(p) {
			a.Permissions = append(a.Permissions, p)
		}
	}
	if rdb := m.Redis; rdb != nil {
		authcache.SetRoleAuth(ctx, rdb, companyID, roleID, a)
	}
	return a
}

// hasPerm 判定操作人是否具备指定权限点之一（wants 为空表示不校验）。
//
// admin 角色短路放行，原因有二：①新增权限点自动对其生效，无需补数据；
// ②避免管理员误配把自己的 role:grant 取消后永久锁死权限配置入口。
func hasPerm(op domain.Operator, wants ...domain.Perm) bool {
	if len(wants) == 0 {
		return true
	}
	if op.RoleCode == domain.RoleCodeAdmin {
		return true
	}
	return domain.HasAnyPerm(op.Permissions, wants...)
}

// requirePerms 认证通过后执行权限判定，不足则 403 并中断请求。
func (m *Middlewares) requirePerms(c *gin.Context, wants ...domain.Perm) {
	if len(wants) == 0 {
		return
	}
	if !hasPerm(GetOperator(c), wants...) {
		response.Fail(c, errs.Forbidden(""))
		c.Abort()
	}
}

// Perm 权限判定中间件（不含认证）。
//
// 用于已由分组级 mw.Auth() 完成认证的路由，在其上追加权限点要求 ——
// 既避免重复执行认证链路（JWT 解析 + 缓存查询），又让权限点就近声明在路由旁：
//
//	r.POST("/grant/:id", mw.Perm(domain.PermRoleGrant), ctl.RoleGrant)
func (m *Middlewares) Perm(perms ...domain.Perm) gin.HandlerFunc {
	return func(c *gin.Context) {
		m.requirePerms(c, perms...)
	}
}

// authenticateStaff 员工令牌统一认证（Auth / StaffAuth 共用）：
// 解析 JWT（锁 HS256）→ jti 黑名单检查 → UserType 白名单 → 画像加载（缓存+DB）
// → 状态校验 → 注入 Operator 到 gin + request context。
// 兼容旧令牌：UserType 为空（签发时无此字段）按员工放行，保持向后兼容。
func (m *Middlewares) authenticateStaff(c *gin.Context) {
	token := extractToken(c)
	if token == "" {
		response.Fail(c, errs.Unauthorized(""))
		c.Abort()
		return
	}
	claims, err := jwtpkg.Parse(m.Cfg.JWT.Secret, m.Cfg.JWT.Issuer, token)
	if err != nil {
		response.Fail(c, errs.Unauthorized(""))
		c.Abort()
		return
	}
	if m.tokenRevoked(c.Request.Context(), claims.ID) {
		response.Fail(c, errs.Unauthorized("登录已失效，请重新登录"))
		c.Abort()
		return
	}
	// 客户令牌（UserType=customer）禁止访问员工接口——防客户 token 撞员工 ID 越权
	if claims.UserType != "" && claims.UserType != jwtpkg.UserTypeStaff {
		response.Fail(c, errs.Unauthorized(""))
		c.Abort()
		return
	}

	u, err := m.loadStaffProfile(c.Request.Context(), claims.UserID)
	if err != nil {
		response.Fail(c, errs.Unauthorized(""))
		c.Abort()
		return
	}
	if u.Status != 1 {
		response.Fail(c, errs.Forbidden("账号已被停用"))
		c.Abort()
		return
	}
	// 加载角色授权（权限点 + 数据范围），供权限判定与行级数据过滤使用
	auth := m.loadRoleAuth(c.Request.Context(), u.CompanyID, u.RoleID)
	op := domain.Operator{
		UserID:      claims.UserID,
		Username:    u.Username,
		Nickname:    u.Nickname,
		CompanyID:   u.CompanyID,
		StoreID:     u.StoreID,
		RoleID:      u.RoleID,
		RoleCode:    domain.RoleCode(auth.RoleCode),
		DataScope:   domain.DataScope(auth.DataScope),
		Permissions: auth.Permissions,
	}
	c.Set(string(OperatorKey), op)
	ctx := context.WithValue(c.Request.Context(), OperatorKey, op)
	c.Request = c.Request.WithContext(ctx)
	c.Next()
}

// Auth JWT 认证中间件（PC 管理后台 / 小程序管理后台），仅接受员工令牌。
//
// perms 为可选的权限点要求（变参）：
//   - 不传 → 仅认证，不做权限判定 —— 与改造前行为完全一致
//   - 传入 → 认证通过后要求至少具备其中一个权限点，否则 403
//
// 变参设计使权限点可以**按批次逐步挂载**到 155 条路由上，
// 无需一次性改动所有调用点，是本次改造风险控制的核心。
func (m *Middlewares) Auth(perms ...domain.Perm) gin.HandlerFunc {
	return func(c *gin.Context) {
		m.authenticateStaff(c)
		if c.IsAborted() {
			return
		}
		m.requirePerms(c, perms...)
	}
}

// StaffAuth 员工 JWT 认证中间件（小程序员工区）：
// 接受 UserType=staff 或旧令牌（空 UserType，向后兼容）。perms 语义同 Auth。
func (m *Middlewares) StaffAuth(perms ...domain.Perm) gin.HandlerFunc {
	return func(c *gin.Context) {
		m.authenticateStaff(c)
		if c.IsAborted() {
			return
		}
		m.requirePerms(c, perms...)
	}
}
