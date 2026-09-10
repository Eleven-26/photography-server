package middleware

import (
	"context"

	"github.com/gin-gonic/gin"

	"photography-server/internal/domain"
	"photography-server/internal/infrastructure"
	"photography-server/internal/model"
	"photography-server/internal/pkg/authcache"
	"photography-server/internal/pkg/errs"
	"photography-server/internal/pkg/jwtpkg"
	"photography-server/internal/response"
)

// ClientUserKey 客户端用户上下文键，同时用于 gin.Context 与 request context
const ClientUserKey ctxKey = "photography.client_user"

// GetClientUser 从 gin 上下文获取当前登录客户（CustomerAuth 注入）
func GetClientUser(c *gin.Context) *service.ClientUser {
	v, _ := c.Get(string(ClientUserKey))
	cu, _ := v.(*service.ClientUser)
	return cu
}

// loadCustomerProfile 加载客户认证画像（优先缓存，回源 DB fail-open），见 auth.go loadStaffProfile 注释
func loadCustomerProfile(ctx context.Context, companyID, customerID int64) (*authcache.CustomerProfile, error) {
	rdb := infrastructure.Redis()
	if rdb != nil {
		if p, hit, err := authcache.GetCustomer(ctx, rdb, companyID, customerID); err == nil && hit {
			return p, nil
		}
	}
	var u model.Customer
	if err := infrastructure.MySQL().WithContext(ctx).Where("company_id = ?", companyID).First(&u, customerID).Error; err != nil {
		return nil, err
	}
	p := &authcache.CustomerProfile{
		Status: int(u.Status),
		Name:   u.Name,
		Mobile: u.Mobile,
	}
	if rdb != nil {
		authcache.SetCustomer(ctx, rdb, companyID, customerID, p)
	}
	return p, nil
}

// CustomerAuth 客户端（H5/小程序）JWT 认证中间件：
// 仅接受 UserType=customer 的令牌，校验客户有效性（含状态，与 Auth/StaffAuth 对齐）并注入 ClientUser 上下文。
func (m *Middlewares) CustomerAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
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
		// 旧令牌 UserType 为空按员工处理，禁止访问客户端接口
		if claims.UserType != jwtpkg.UserTypeCustomer {
			response.Fail(c, errs.Unauthorized(""))
			c.Abort()
			return
		}
		u, err := loadCustomerProfile(c.Request.Context(), claims.CompanyID, claims.UserID)
		if err != nil {
			response.Fail(c, errs.Unauthorized(""))
			c.Abort()
			return
		}
		// 客户状态校验（#28）：与 Auth/StaffAuth 的 Status 检查对齐——被停用/流失的客户禁止访问
		if u.Status != 2 {
			response.Fail(c, errs.Forbidden("账号不可用，请联系工作室"))
			c.Abort()
			return
		}
		cu := &domain.ClientUser{
			CustomerID: claims.UserID,
			CompanyID:  claims.CompanyID,
			Mobile:     u.Mobile,
			Name:       u.Name,
		}
		c.Set(string(ClientUserKey), cu)
		ctx := context.WithValue(c.Request.Context(), ClientUserKey, cu)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
