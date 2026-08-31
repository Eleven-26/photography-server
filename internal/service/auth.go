package service

import (
	"time"

	"golang.org/x/crypto/bcrypt"

	"photography-server/internal/model"
	"photography-server/internal/pkg/errs"
	"photography-server/internal/pkg/jwtpkg"
)

type LoginReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type LoginResp struct {
	Token string        `json:"token"`
	User  model.SysUser `json:"user"`
}

func (s *Service) Login(secret, issuer string, expireHours int, req LoginReq, ip string) (*LoginResp, error) {
	var u model.SysUser
	if err := s.DB().Where("username = ?", req.Username).First(&u).Error; err != nil {
		return nil, errs.BadRequest("账号或密码错误")
	}
	if u.Status != 1 {
		return nil, errs.Forbidden("账号已被停用")
	}
	if bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(req.Password)) != nil {
		return nil, errs.BadRequest("账号或密码错误")
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

	now := time.Now().Format("2006-01-02 15:04:05")
	s.DB().Model(&u).Updates(map[string]interface{}{
		"last_login_at": now,
		"last_login_ip": ip,
	})
	u.Password = ""
	return &LoginResp{Token: token, User: u}, nil
}

func (s *Service) Profile(op Operator) (*model.SysUser, error) {
	var u model.SysUser
	if err := s.tenant(op).First(&u, op.UserID).Error; err != nil {
		return nil, errs.NotFound("用户不存在")
	}
	u.Password = ""
	return &u, nil
}

func (s *Service) ChangePassword(op Operator, oldPwd, newPwd string) error {
	var u model.SysUser
	if err := s.tenant(op).First(&u, op.UserID).Error; err != nil {
		return errs.NotFound("用户不存在")
	}
	if bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(oldPwd)) != nil {
		return errs.BadRequest("原密码错误")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPwd), bcrypt.DefaultCost)
	if err != nil {
		return errs.Internal("")
	}
	return s.tenant(op).Model(&u).Update("password", string(hash)).Error
}
