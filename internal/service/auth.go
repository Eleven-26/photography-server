package service

import (
	"golang.org/x/crypto/bcrypt"

	"photography-server/internal/common"
	"photography-server/internal/model"
	"photography-server/internal/pkg/errs"
	"photography-server/internal/pkg/jwtpkg"
	"photography-server/internal/presentation/dto"
)

func (s *Service) Login(secret, issuer string, expireHours int, req dto.LoginReq, ip string) (*dto.LoginResp, error) {
	u, err := s.AuthRepo.GetByUsername(req.Username)
	if err != nil {
		return nil, errs.BadRequest(common.ErrAccountWrong)
	}
	if u.Status != 1 {
		return nil, errs.Forbidden(common.ErrAccountDisabled)
	}
	if bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(req.Password)) != nil {
		return nil, errs.BadRequest(common.ErrAccountWrong)
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

	s.AuthRepo.UpdateLoginInfo(u.ID, ip)
	u.Password = ""
	return &dto.LoginResp{Token: token, User: *u}, nil
}

func (s *Service) Profile(op Operator) (*model.SysUser, error) {
	u, err := s.AuthRepo.GetByID(op.CompanyID, op.UserID)
	if err != nil {
		return nil, errs.NotFound(common.ErrUserNotFound)
	}
	u.Password = ""
	return u, nil
}

func (s *Service) ChangePassword(op Operator, oldPwd, newPwd string) error {
	u, err := s.AuthRepo.GetByID(op.CompanyID, op.UserID)
	if err != nil {
		return errs.NotFound(common.ErrUserNotFound)
	}
	if bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(oldPwd)) != nil {
		return errs.BadRequest(common.ErrPasswordWrong)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPwd), bcrypt.DefaultCost)
	if err != nil {
		return errs.Internal("")
	}
	return s.AuthRepo.UpdatePassword(op.UserID, string(hash))
}
