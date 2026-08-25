package service

import (
	"golang.org/x/crypto/bcrypt"

	"photography-server/internal/model"
	"photography-server/internal/pkg/errs"
)

type UserCreateReq struct {
	StoreID  int64  `json:"store_id"`
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Nickname string `json:"nickname"`
	Mobile   string `json:"mobile"`
	RoleID   int64  `json:"role_id" binding:"required"`
	Status   int    `json:"status"`
}

type UserUpdateReq struct {
	StoreID  int64  `json:"store_id"`
	Nickname string `json:"nickname"`
	Mobile   string `json:"mobile"`
	Email    string `json:"email"`
	Avatar   string `json:"avatar"`
	RoleID   int64  `json:"role_id"`
	Status   int    `json:"status"`
}

func (s *Service) ListUsers(op Operator, page, pageSize int, keyword string, storeID int64) ([]model.SysUser, int64, error) {
	q := s.tenant(op)
	if keyword != "" {
		kw := "%" + keyword + "%"
		q = q.Where("username LIKE ? OR nickname LIKE ? OR mobile LIKE ?", kw, kw, kw)
	}
	if storeID > 0 {
		q = q.Where("store_id = ?", storeID)
	}
	var total int64
	if err := q.Model(&model.SysUser{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.SysUser
	page, pageSize = normalizePage(page, pageSize)
	if err := q.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	for i := range list {
		list[i].Password = ""
	}
	return list, total, nil
}

func (s *Service) CreateUser(op Operator, req UserCreateReq) error {
	var count int64
	s.tenant(op).Model(&model.SysUser{}).Where("username = ?", req.Username).Count(&count)
	if count > 0 {
		return errs.Conflict("登录账号已存在")
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
	return s.tenant(op).Create(&u).Error
}

func (s *Service) UpdateUser(op Operator, id int64, req UserUpdateReq) error {
	var u model.SysUser
	if err := s.tenant(op).First(&u, id).Error; err != nil {
		return errs.NotFound("用户不存在")
	}
	return s.tenant(op).Model(&u).Updates(map[string]interface{}{
		"store_id":   req.StoreID,
		"nickname":   req.Nickname,
		"mobile":     req.Mobile,
		"email":      req.Email,
		"avatar":     req.Avatar,
		"role_id":    req.RoleID,
		"status":     req.Status,
		"updated_by": op.UserID,
	}).Error
}

func (s *Service) DeleteUser(op Operator, id int64) error {
	if id == op.UserID {
		return errs.BadRequest("不能删除当前登录账号")
	}
	return s.tenant(op).Delete(&model.SysUser{}, id).Error
}

func (s *Service) ResetPassword(op Operator, id int64, pwd string) error {
	var u model.SysUser
	if err := s.tenant(op).First(&u, id).Error; err != nil {
		return errs.NotFound("用户不存在")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.DefaultCost)
	if err != nil {
		return errs.Internal("")
	}
	return s.tenant(op).Model(&u).Update("password", string(hash)).Error
}

// -------- 角色 --------

func (s *Service) ListRoles(op Operator) ([]model.SysRole, error) {
	var list []model.SysRole
	err := s.tenant(op).Order("id ASC").Find(&list).Error
	return list, err
}

func (s *Service) CreateRole(op Operator, name, code, remark string) error {
	r := model.SysRole{
		TenantBase: model.TenantBase{
			Base:      model.Base{CreatedBy: op.UserID, UpdatedBy: op.UserID},
			CompanyID: op.CompanyID,
		},
		Name:   name,
		Code:   code,
		Remark: remark,
		Status: 1,
	}
	return s.tenant(op).Create(&r).Error
}

func (s *Service) UpdateRole(op Operator, id int64, name, code, remark string, status int) error {
	var r model.SysRole
	if err := s.tenant(op).First(&r, id).Error; err != nil {
		return errs.NotFound("角色不存在")
	}
	return s.tenant(op).Model(&r).Updates(map[string]interface{}{
		"name": name, "code": code, "remark": remark, "status": status, "updated_by": op.UserID,
	}).Error
}

func (s *Service) DeleteRole(op Operator, id int64) error {
	return s.tenant(op).Delete(&model.SysRole{}, id).Error
}

// -------- 门店 --------

func (s *Service) ListStores(op Operator) ([]model.SysStore, error) {
	var list []model.SysStore
	err := s.tenant(op).Order("id ASC").Find(&list).Error
	return list, err
}

func (s *Service) CreateStore(op Operator, name, address, phone string) error {
	st := model.SysStore{
		TenantBase: model.TenantBase{
			Base:      model.Base{CreatedBy: op.UserID, UpdatedBy: op.UserID},
			CompanyID: op.CompanyID,
		},
		Name: name, Address: address, Phone: phone, Status: 1,
	}
	return s.tenant(op).Create(&st).Error
}

func (s *Service) UpdateStore(op Operator, id int64, name, address, phone string, status int) error {
	var st model.SysStore
	if err := s.tenant(op).First(&st, id).Error; err != nil {
		return errs.NotFound("门店不存在")
	}
	return s.tenant(op).Model(&st).Updates(map[string]interface{}{
		"name": name, "address": address, "phone": phone, "status": status, "updated_by": op.UserID,
	}).Error
}

func (s *Service) DeleteStore(op Operator, id int64) error {
	return s.tenant(op).Delete(&model.SysStore{}, id).Error
}

func normalizePage(page, pageSize int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 200 {
		pageSize = 200
	}
	return page, pageSize
}
