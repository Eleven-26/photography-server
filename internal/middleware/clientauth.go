package middleware

import (
	"context"

	"github.com/gin-gonic/gin"

	"photography-server/internal/infrastructure"
	"photography-server/internal/model"
	"photography-server/internal/pkg/errs"
	"photography-server/internal/pkg/jwtpkg"
	"photography-server/internal/response"
	"photography-server/internal/service"
)

// ClientUserKey 客户端用户上下文键，同时用于 gin.Context 与 request context
const ClientUserKey ctxKey = "photography.client_user"

// GetClientUser 从 gin 上下文获取当前登录客户（CustomerAuth 注入）
func GetClientUser(c *gin.Context) *service.ClientUser {
	v, _ := c.Get(string(ClientUserKey))
	cu, _ := v.(*service.ClientUser)
	return cu
}

// CustomerAuth 客户端（H5/小程序）JWT 认证中间件：
// 仅接受 UserType=customer 的令牌，校验客户有效性并注入 ClientUser 上下文。
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
		// 旧令牌 UserType 为空按员工处理，禁止访问客户端接口
		if claims.UserType != jwtpkg.UserTypeCustomer {
			response.Fail(c, errs.Unauthorized(""))
			c.Abort()
			return
		}
		var u model.Customer
		if err := infrastructure.MySQL().Where("company_id = ?", claims.CompanyID).First(&u, claims.UserID).Error; err != nil {
			response.Fail(c, errs.Unauthorized(""))
			c.Abort()
			return
		}
		cu := &service.ClientUser{
			CustomerID: u.ID,
			CompanyID:  u.CompanyID,
			Mobile:     u.Mobile,
			Name:       u.Name,
		}
		c.Set(string(ClientUserKey), cu)
		ctx := context.WithValue(c.Request.Context(), ClientUserKey, cu)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

// StaffAuth 员工 JWT 认证中间件（摄影师 App）：
// 接受 UserType=staff 或旧令牌（空 UserType，向后兼容），注入 Operator 上下文。
func (m *Middlewares) StaffAuth() gin.HandlerFunc {
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
		if claims.UserType != "" && claims.UserType != jwtpkg.UserTypeStaff {
			response.Fail(c, errs.Unauthorized(""))
			c.Abort()
			return
		}
		var u model.SysUser
		if err := infrastructure.MySQL().First(&u, claims.UserID).Error; err != nil {
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
			UserID:    u.ID,
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
}
