package service

import (
	"golang.org/x/crypto/bcrypt"

	"photography-server/internal/model"
	"photography-server/internal/pkg/errs"
	"photography-server/internal/presentation/dto"
)

func (s *Service) ListUsers(op Operator, page, pageSize int, keyword string, storeID int64) ([]model.SysUser, int64, error) {
	list, total, err := s.UserRepo.List(op.CompanyID, page, pageSize, keyword, storeID)
	if err != nil {
		return nil, 0, err
	}
	for i := range list {
		list[i].Password = ""
	}
	return list, total, nil
}

func (s *Service) CreateUser(op Operator, req dto.UserCreateReq) error {
	count, _ := s.UserRepo.CountByUsername(op.CompanyID, req.Username)
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
	return s.UserRepo.Create(&u)
}

func (s *Service) UpdateUser(op Operator, id int64, req dto.UserUpdateReq) error {
	_, err := s.UserRepo.GetByID(op.CompanyID, id)
	if err != nil {
		return errs.NotFound(errs.ErrUserNotFound)
	}
	return s.UserRepo.Update(op.CompanyID, id, map[string]interface{}{
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

func (s *Service) DeleteUser(op Operator, id int64) error {
	if id == op.UserID {
		return errs.BadRequest(errs.ErrUserSelfDelete)
	}
	return s.UserRepo.Delete(op.CompanyID, id)
}

func (s *Service) ResetPassword(op Operator, id int64, pwd string) error {
	_, err := s.UserRepo.GetByID(op.CompanyID, id)
	if err != nil {
		return errs.NotFound(errs.ErrUserNotFound)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.DefaultCost)
	if err != nil {
		return errs.Internal("")
	}
	return s.UserRepo.UpdatePassword(op.CompanyID, id, string(hash))
}

// -------- 角色 --------

func (s *Service) ListRoles(op Operator) ([]model.SysRole, error) {
	return s.UserRepo.ListRoles(op.CompanyID)
}

func (s *Service) CreateRole(op Operator, req dto.RoleCreateReq) error {
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
	return s.UserRepo.CreateRole(&r)
}

func (s *Service) UpdateRole(op Operator, id int64, req dto.RoleUpdateReq) error {
	return s.UserRepo.UpdateRole(op.CompanyID, id, map[string]interface{}{
		"name": req.Name, "code": req.Code, "remark": req.Remark, "status": req.Status, "updated_by": op.UserID,
	})
}

func (s *Service) DeleteRole(op Operator, id int64) error {
	return s.UserRepo.DeleteRole(op.CompanyID, id)
}

// -------- 门店 --------

func (s *Service) ListStores(op Operator) ([]model.SysStore, error) {
	return s.UserRepo.ListStores(op.CompanyID)
}

func (s *Service) CreateStore(op Operator, req dto.StoreCreateReq) error {
	st := model.SysStore{
		TenantBase: model.TenantBase{
			Base:      model.Base{CreatedBy: op.UserID, UpdatedBy: op.UserID},
			CompanyID: op.CompanyID,
		},
		Name: req.Name, Address: req.Address, Phone: req.Phone, Status: 1,
	}
	return s.UserRepo.CreateStore(&st)
}

func (s *Service) UpdateStore(op Operator, id int64, req dto.StoreUpdateReq) error {
	return s.UserRepo.UpdateStore(op.CompanyID, id, map[string]interface{}{
		"name": req.Name, "address": req.Address, "phone": req.Phone, "status": req.Status, "updated_by": op.UserID,
	})
}

func (s *Service) DeleteStore(op Operator, id int64) error {
	return s.UserRepo.DeleteStore(op.CompanyID, id)
}
