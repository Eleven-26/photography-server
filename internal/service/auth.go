package service

import (
	"context"
	"time"

	"golang.org/x/crypto/bcrypt"

	"photography-server/internal/domain"
	"photography-server/internal/model"
	"photography-server/internal/pkg/errs"
	"photography-server/internal/pkg/jwtpkg"
	"photography-server/internal/presentation/dto"
)

// Login 登录。ctx 由 controller 传入（c.Request.Context()），透传给 repo 使 SQL 挂到当前链路。
// 修复 #45.2：添加失败限流，防止暴力破解
func (s *Service) Login(ctx context.Context, secret, issuer string, expireHours int, req dto.LoginReq, ip string) (*dto.LoginResp, error) {
	rdb := s.redis()

	// 检查 IP 是否被锁定（失败 10 次）
	if rdb != nil {
		ipLockKey := "login:ip:" + ip + "_locked"
		if locked, _ := rdb.Exists(ctx, ipLockKey).Result(); locked > 0 {
			return nil, errs.BadRequest("登录失败次数过多，请 15 分钟后再试")
		}
	}

	// 检查用户名是否被锁定（失败 5 次）
	if rdb != nil {
		userLockKey := "login:user:" + req.Username + "_locked"
		if locked, _ := rdb.Exists(ctx, userLockKey).Result(); locked > 0 {
			return nil, errs.BadRequest("账号已被锁定，请 15 分钟后再试")
		}
	}

	u, err := s.AuthRepo.GetByUsername(ctx, req.Username)
	if err != nil {
		// 统一错误消息，不泄露账号是否存在
		return nil, errs.BadRequest(errs.ErrAccountWrong)
	}

	// 账号停用也返回统一错误，避免泄露账号状态
	if u.Status != 1 {
		return nil, errs.BadRequest(errs.ErrAccountWrong)
	}

	if bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(req.Password)) != nil {
		// 密码错误，记录失败次数
		if rdb != nil {
			// IP 维度计数
			ipKey := "login:ip:" + ip
			ipCount, _ := rdb.Incr(ctx, ipKey).Result()
			if ipCount == 1 {
				rdb.Expire(ctx, ipKey, 15*time.Minute)
			}
			if ipCount >= 10 {
				rdb.Set(ctx, ipKey+"_locked", "1", 15*time.Minute)
			}

			// 用户名维度计数
			userKey := "login:user:" + req.Username
			userCount, _ := rdb.Incr(ctx, userKey).Result()
			if userCount == 1 {
				rdb.Expire(ctx, userKey, 15*time.Minute)
			}
			if userCount >= 5 {
				rdb.Set(ctx, userKey+"_locked", "1", 15*time.Minute)
			}
		}
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

	// 登录成功，清除失败计数
	if rdb != nil {
		rdb.Del(ctx, "login:ip:"+ip)
		rdb.Del(ctx, "login:user:"+req.Username)
	}

	u.Password = ""
	return &dto.LoginResp{Token: token, User: s.buildUserInfo(ctx, u)}, nil
}

func (s *Service) Profile(ctx context.Context, op Operator) (*dto.UserInfoVO, error) {
	u, err := s.AuthRepo.GetByID(ctx, op.CompanyID, op.UserID)
	if err != nil {
		return nil, errs.NotFound(errs.ErrUserNotFound)
	}
	u.Password = ""
	vo := s.buildUserInfo(ctx, u)
	return &vo, nil
}

// buildUserInfo 组装登录 / 个人资料返回的用户信息：SysUser + 角色 + 权限 + 数据范围。
//
// 前端权限判定依赖 role_code 与 permissions，缺失会导致按钮与路由控制整体失效
// （改造前这两个字段就从未下发，前端 hasPerm 因此形同虚设）。
//
// admin 角色在此补齐全量权限点：后端判定靠角色码短路，但前端渲染菜单/按钮需要
// 具体清单，否则管理员界面会被自身权限判定误隐藏。
func (s *Service) buildUserInfo(ctx context.Context, u *model.SysUser) dto.UserInfoVO {
	vo := dto.UserInfoVO{SysUser: *u, Permissions: []string{}}
	if u.RoleID <= 0 {
		return vo
	}
	if role, err := s.UserRepo.GetRoleByID(ctx, u.CompanyID, u.RoleID); err == nil {
		vo.RoleCode = role.Code
		vo.RoleName = role.Name
		vo.DataScope = role.DataScope
	}
	if perms, err := s.UserRepo.ListPermsByRole(ctx, u.CompanyID, u.RoleID); err == nil {
		valid := make([]string, 0, len(perms))
		for _, p := range perms {
			if domain.IsValidPerm(p) {
				valid = append(valid, p)
			}
		}
		vo.Permissions = valid
	}
	if domain.RoleCode(vo.RoleCode) == domain.RoleCodeAdmin {
		all := domain.AllPermissions()
		flat := make([]string, 0, len(all))
		for _, p := range all {
			flat = append(flat, string(p))
		}
		vo.Permissions = flat
		vo.DataScope = int(domain.ScopeAll)
	}
	return vo
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
	s.invalidateStaffCache(ctx, op.UserID)
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
	rdb := s.redis()
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
