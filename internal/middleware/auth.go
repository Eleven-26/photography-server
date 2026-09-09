package middleware

import (
	"context"

	"github.com/gin-gonic/gin"

	"photography-server/internal/infrastructure"
	"photography-server/internal/model"
	"photography-server/internal/pkg/authcache"
	"photography-server/internal/pkg/errs"
	"photography-server/internal/pkg/jwtpkg"
	"photography-server/internal/response"
	"photography-server/internal/service"
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
// Redis 不可用（rdb==nil）或查询出错时 fail-open 放行——吊销依赖 Redis 是尽力而为；
// 登录链路本身强依赖 Redis，Redis 长期不可用时服务已按 fail-fast 拒启（见 main.go）。
func tokenRevoked(ctx context.Context, jti string) bool {
	if jti == "" {
		return false // 旧令牌无 jti，无吊销能力
	}
	rdb := infrastructure.Redis()
	if rdb == nil {
		return false
	}
	n, err := rdb.Exists(ctx, jwtpkg.BlacklistKey(jti)).Result()
	return err == nil && n > 0
}

// loadStaffProfile 加载员工认证画像：优先 Redis 缓存（60s TTL），未命中回源 DB。
// 缓存命中但画像不全时同样回源 DB 覆盖（fail-open，保证租户/角色字段正确）。
func loadStaffProfile(ctx context.Context, userID int64) (*authcache.StaffProfile, error) {
	rdb := infrastructure.Redis()
	if rdb != nil {
		if p, hit, err := authcache.GetStaff(ctx, rdb, userID); err == nil && hit {
			return p, nil
		}
	}
	var u model.SysUser
	if err := infrastructure.MySQL().WithContext(ctx).First(&u, userID).Error; err != nil {
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
	if tokenRevoked(c.Request.Context(), claims.ID) {
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

	u, err := loadStaffProfile(c.Request.Context(), claims.UserID)
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
	op := service.Operator{
		UserID:    claims.UserID,
		Username:  u.Username,
		Nickname:  u.Nickname,
		CompanyID: u.CompanyID,
		StoreID:   u.StoreID,
		RoleID:    u.RoleID,
	}
	c.Set(string(OperatorKey), op)
	ctx := context.WithValue(c.Request.Context(), OperatorKey, op)
	c.Request = c.Request.WithContext(ctx)
	c.Next()
}

// Auth JWT 认证中间件（PC 管理后台 / 小程序管理后台），仅接受员工令牌
func (m *Middlewares) Auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		m.authenticateStaff(c)
	}
}

// StaffAuth 员工 JWT 认证中间件（小程序员工区）：
// 接受 UserType=staff 或旧令牌（空 UserType，向后兼容）
func (m *Middlewares) StaffAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		m.authenticateStaff(c)
	}
}
