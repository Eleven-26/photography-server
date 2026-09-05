package service

import (
	"context"

	"golang.org/x/crypto/bcrypt"

	"photography-server/internal/model"
	"photography-server/internal/pkg/errs"
	"photography-server/internal/presentation/dto"
)

func (s *Service) ListUsers(ctx context.Context, op Operator, page, pageSize int, keyword string, storeID int64) ([]model.SysUser, int64, error) {
	list, total, err := s.UserRepo.List(ctx, op.CompanyID, page, pageSize, keyword, storeID)
	if err != nil {
		return nil, 0, err
	}
	for i := range list {
		list[i].Password = ""
	}
	return list, total, nil
}

func (s *Service) CreateUser(ctx context.Context, op Operator, req dto.UserCreateReq) error {
	count, _ := s.UserRepo.CountByUsername(ctx, op.CompanyID, req.Username)
	if count > 0 {
		return errs.Conflict(errs.ErrUserDuplicate)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return errs.Internal("")
	}
	u := model.SysUser{
		TenantBase: model.TenantBase{
			Base:      model.Base{CreatedBy: op.UserID, UpdatedBy: op.UserID},
			CompanyID: op.CompanyID,
		},
		StoreID:  req.StoreID,
		Username: req.Username,
		Password: string(hash),
		Nickname: req.Nickname,
		Mobile:   req.Mobile,
		RoleID:   req.RoleID,
		Status:   req.Status,
	}
	if u.Status == 0 {
		u.Status = 1
	}
	return s.UserRepo.Create(ctx, &u)
}

func (s *Service) UpdateUser(ctx context.Context, op Operator, id int64, req dto.UserUpdateReq) error {
	_, err := s.UserRepo.GetByID(ctx, op.CompanyID, id)
	if err != nil {
		return errs.NotFound(errs.ErrUserNotFound)
	}
	return s.UserRepo.Update(ctx, op.CompanyID, id, map[string]interface{}{
		"store_id":   req.StoreID,
		"nickname":   req.Nickname,
		"mobile":     req.Mobile,
		"email":      req.Email,
		"avatar":     req.Avatar,
		"role_id":    req.RoleID,
		"status":     req.Status,
		"updated_by": op.UserID,
	})
}

func (s *Service) DeleteUser(ctx context.Context, op Operator, id int64) error {
	if id == op.UserID {
		return errs.BadRequest(errs.ErrUserSelfDelete)
	}
	return s.UserRepo.Delete(ctx, op.CompanyID, id)
}

func (s *Service) ResetPassword(ctx context.Context, op Operator, id int64, pwd string) error {
	_, err := s.UserRepo.GetByID(ctx, op.CompanyID, id)
	if err != nil {
		return errs.NotFound(errs.ErrUserNotFound)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.DefaultCost)
	if err != nil {
		return errs.Internal("")
	}
	return s.UserRepo.UpdatePassword(ctx, op.CompanyID, id, string(hash))
}

// -------- 角色 --------

func (s *Service) ListRoles(ctx context.Context, op Operator) ([]model.SysRole, error) {
	return s.UserRepo.ListRoles(ctx, op.CompanyID)
}

func (s *Service) CreateRole(ctx context.Context, op Operator, req dto.RoleCreateReq) error {
	r := model.SysRole{
		TenantBase: model.TenantBase{
			Base:      model.Base{CreatedBy: op.UserID, UpdatedBy: op.UserID},
			CompanyID: op.CompanyID,
		},
		Name:   req.Name,
		Code:   req.Code,
		Remark: req.Remark,
		Status: 1,
	}
	return s.UserRepo.CreateRole(ctx, &r)
}

func (s *Service) UpdateRole(ctx context.Context, op Operator, id int64, req dto.RoleUpdateReq) error {
	return s.UserRepo.UpdateRole(ctx, op.CompanyID, id, map[string]interface{}{
		"name": req.Name, "code": req.Code, "remark": req.Remark, "status": req.Status, "updated_by": op.UserID,
	})
}

func (s *Service) DeleteRole(ctx context.Context, op Operator, id int64) error {
	return s.UserRepo.DeleteRole(ctx, op.CompanyID, id)
}

// -------- 门店 --------

func (s *Service) ListStores(ctx context.Context, op Operator) ([]model.SysStore, error) {
	return s.UserRepo.ListStores(ctx, op.CompanyID)
}

func (s *Service) CreateStore(ctx context.Context, op Operator, req dto.StoreCreateReq) error {
	st := model.SysStore{
		TenantBase: model.TenantBase{
			Base:      model.Base{CreatedBy: op.UserID, UpdatedBy: op.UserID},
			CompanyID: op.CompanyID,
		},
		Name: req.Name, Address: req.Address, Phone: req.Phone, Status: 1,
	}
	return s.UserRepo.CreateStore(ctx, &st)
}

func (s *Service) UpdateStore(ctx context.Context, op Operator, id int64, req dto.StoreUpdateReq) error {
	return s.UserRepo.UpdateStore(ctx, op.CompanyID, id, map[string]interface{}{
		"name": req.Name, "address": req.Address, "phone": req.Phone, "status": req.Status, "updated_by": op.UserID,
	})
}

func (s *Service) DeleteStore(ctx context.Context, op Operator, id int64) error {
	return s.UserRepo.DeleteStore(ctx, op.CompanyID, id)
}
