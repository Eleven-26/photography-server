package service

import (
	"context"
	"time"

	"golang.org/x/crypto/bcrypt"

	"photography-server/internal/infrastructure"
	"photography-server/internal/model"
	"photography-server/internal/pkg/errs"
	"photography-server/internal/pkg/jwtpkg"
	"photography-server/internal/presentation/dto"
)

// Login 登录。ctx 由 controller 传入（c.Request.Context()），透传给 repo 使 SQL 挂到当前链路。
func (s *Service) Login(ctx context.Context, secret, issuer string, expireHours int, req dto.LoginReq, ip string) (*dto.LoginResp, error) {
	u, err := s.AuthRepo.GetByUsername(ctx, req.Username)
	if err != nil {
		return nil, errs.BadRequest(errs.ErrAccountWrong)
	}
	if u.Status != 1 {
		return nil, errs.Forbidden(errs.ErrAccountDisabled)
	}
	if bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(req.Password)) != nil {
		return nil, errs.BadRequest(errs.ErrAccountWrong)
	}

	token, err := jwtpkg.Generate(secret, issuer, expireHours, jwtpkg.Claims{
		UserID:    u.ID,
		Username:  u.Username,
		CompanyID: u.CompanyID,
		StoreID:   u.StoreID,
		RoleID:    u.RoleID,
	})
	if err != nil {
		return nil, errs.Internal("")
	}

	s.AuthRepo.UpdateLoginInfo(ctx, u.ID, ip)
	u.Password = ""
	return &dto.LoginResp{Token: token, User: *u}, nil
}

func (s *Service) Profile(ctx context.Context, op Operator) (*model.SysUser, error) {
	u, err := s.AuthRepo.GetByID(ctx, op.CompanyID, op.UserID)
	if err != nil {
		return nil, errs.NotFound(errs.ErrUserNotFound)
	}
	u.Password = ""
	return u, nil
}

func (s *Service) ChangePassword(ctx context.Context, op Operator, oldPwd, newPwd string) error {
	u, err := s.AuthRepo.GetByID(ctx, op.CompanyID, op.UserID)
	if err != nil {
		return errs.NotFound(errs.ErrUserNotFound)
	}
	if bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(oldPwd)) != nil {
		return errs.BadRequest(errs.ErrPasswordWrong)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPwd), bcrypt.DefaultCost)
	if err != nil {
		return errs.Internal("")
	}
	if err := s.AuthRepo.UpdatePassword(ctx, op.UserID, string(hash)); err != nil {
		return err
	}
	// 改密后立即使该用户所有已签发令牌失效（认证画像缓存 60s 内仍可能放行旧会话，
	// 主动删除让"改密=踢下线"即时生效）
	invalidateStaffCache(ctx, op.UserID)
	return nil
}

// Logout 登出：把令牌 jti 写入黑名单，使其立即失效（替代"无状态 JWT 登出仅返回成功"）。
// 旧令牌无 jti 无法吊销，靠过期自然失效；Redis 不可用时尽力而为（fail-open，同中间件策略）。
func (s *Service) Logout(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	claims, err := jwtpkg.Parse(s.JWTSecret, s.JWTIssuer, token)
	if err != nil {
		return nil // 无效令牌视为已登出
	}
	if claims.ID == "" {
		return nil
	}
	rdb := infrastructure.Redis()
	if rdb == nil {
		return nil
	}
	ttl := time.Until(claims.ExpiresAt.Time)
	if ttl <= 0 {
		return nil
	}
	// 写失败返回错误：登出动作未完成，前端应提示重试（避免"登出成功但 token 仍有效"）
	if err := rdb.Set(ctx, jwtpkg.BlacklistKey(claims.ID), claims.UserID, ttl).Err(); err != nil {
		return errs.Internal("")
	}
	return nil
}
