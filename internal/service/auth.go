package service

import (
	"golang.org/x/crypto/bcrypt"

	"photography-server/internal/model"
	"photography-server/internal/pkg/errs"
	"photography-server/internal/pkg/jwtpkg"
	"photography-server/internal/presentation/dto"
)

func (s *Service) Login(secret, issuer string, expireHours int, req dto.LoginReq, ip string) (*dto.LoginResp, error) {
	u, err := s.AuthRepo.GetByUsername(req.Username)
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

	s.AuthRepo.UpdateLoginInfo(u.ID, ip)
	u.Password = ""
	return &dto.LoginResp{Token: token, User: *u}, nil
}

func (s *Service) Profile(op Operator) (*model.SysUser, error) {
	u, err := s.AuthRepo.GetByID(op.CompanyID, op.UserID)
	if err != nil {
		return nil, errs.NotFound(errs.ErrUserNotFound)
	}
	u.Password = ""
	return u, nil
}

func (s *Service) ChangePassword(op Operator, oldPwd, newPwd string) error {
	u, err := s.AuthRepo.GetByID(op.CompanyID, op.UserID)
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
	return s.AuthRepo.UpdatePassword(op.UserID, string(hash))
}
