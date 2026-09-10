package middleware

import (
	"github.com/gin-gonic/gin"

	"photography-server/internal/pkg/errs"
	"photography-server/internal/pkg/jwtpkg"
	"photography-server/internal/presentation/response"
)

// AssetAuth 静态资源（/uploads）访问鉴权中间件（#9）：
// 样片/精修成片/原片/收款退款凭证/头像均为客户隐私数据，禁止匿名访问与枚举。
// 任一端（员工 staff 或客户 customer）的有效令牌均可访问——上传内容既有员工侧也有客户侧；
// 与 Auth/StaffAuth/CustomerAuth 同一套策略：锁 HS256 + jti 登出黑名单 + 画像缓存 + 状态校验。
// 订单级归属/有效期校验需要文件-订单映射，作为后续增强（静态目录粒度无法做行级鉴权）。
func (m *Middlewares) AssetAuth() gin.HandlerFunc {
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
		if m.tokenRevoked(c.Request.Context(), claims.ID) {
			response.Fail(c, errs.Unauthorized("登录已失效，请重新登录"))
			c.Abort()
			return
		}

		// 员工令牌（含旧版无 utype 的向后兼容）：校验员工存在且启用
		if claims.UserType == "" || claims.UserType == jwtpkg.UserTypeStaff {
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
			c.Next()
			return
		}
		// 客户令牌：校验客户存在且未流失（2-活跃）
		if claims.UserType == jwtpkg.UserTypeCustomer {
			u, err := m.loadCustomerProfile(c.Request.Context(), claims.CompanyID, claims.UserID)
			if err != nil {
				response.Fail(c, errs.Unauthorized(""))
				c.Abort()
				return
			}
			if u.Status != 2 {
				response.Fail(c, errs.Forbidden("账号状态异常"))
				c.Abort()
				return
			}
			c.Next()
			return
		}
		response.Fail(c, errs.Unauthorized(""))
		c.Abort()
	}
}
