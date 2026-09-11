package service

import (
	"context"
	"fmt"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"photography-server/internal/domain"
	"photography-server/internal/model"
	"photography-server/internal/pkg/authcache"
	"photography-server/internal/pkg/errs"
	"photography-server/internal/presentation/dto"
	"photography-server/internal/repository"
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
	if err := s.UserRepo.Update(ctx, op.CompanyID, id, map[string]interface{}{
		"store_id":   req.StoreID,
		"nickname":   req.Nickname,
		"mobile":     req.Mobile,
		"email":      req.Email,
		"avatar":     req.Avatar,
		"role_id":    req.RoleID,
		"status":     req.Status,
		"updated_by": op.UserID,
	}); err != nil {
		return err
	}
	// 状态/角色可能变更：失效认证画像缓存，停用即时生效（#30）
	s.invalidateStaffCache(ctx, id)
	return nil
}

func (s *Service) DeleteUser(ctx context.Context, op Operator, id int64) error {
	if id == op.UserID {
		return errs.BadRequest(errs.ErrUserSelfDelete)
	}
	if err := s.UserRepo.Delete(ctx, op.CompanyID, id); err != nil {
		return err
	}
	s.invalidateStaffCache(ctx, id)
	return nil
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
	if err := s.UserRepo.UpdatePassword(ctx, op.CompanyID, id, string(hash)); err != nil {
		return err
	}
	s.invalidateStaffCache(ctx, id)
	return nil
}

// -------- 角色 --------

// ListRoles 角色列表（附带各角色已配置的权限点数量，供角色管理界面展示）
func (s *Service) ListRoles(ctx context.Context, op Operator) ([]model.SysRole, error) {
	list, err := s.UserRepo.ListRoles(ctx, op.CompanyID)
	if err != nil {
		return nil, err
	}
	counts, err := s.UserRepo.CountPermsByRoles(ctx, op.CompanyID)
	if err != nil {
		return nil, err
	}
	for i := range list {
		list[i].PermissionCount = counts[list[i].ID]
	}
	return list, nil
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
		// 新角色默认「全部数据」：与存量角色一致，管理员在权限配置界面显式收窄
		DataScope: int(domain.ScopeAll),
	}
	return s.UserRepo.CreateRole(ctx, &r)
}

func (s *Service) UpdateRole(ctx context.Context, op Operator, id int64, req dto.RoleUpdateReq) error {
	if err := s.UserRepo.UpdateRole(ctx, op.CompanyID, id, map[string]interface{}{
		"name": req.Name, "code": req.Code, "remark": req.Remark, "status": req.Status, "updated_by": op.UserID,
	}); err != nil {
		return err
	}
	// 角色编码参与 admin 短路判定，变更后需清缓存即时生效
	s.invalidateRoleAuth(ctx, op.CompanyID, id)
	return nil
}

// DeleteRole 删除角色。占用校验：角色下仍有成员时必须先转移，
// 否则成员会因角色消失而失去全部权限（角色查询失败 → 空权限集）。
func (s *Service) DeleteRole(ctx context.Context, op Operator, id int64) error {
	n, err := s.UserRepo.CountUsersByRole(ctx, op.CompanyID, id)
	if err != nil {
		return err
	}
	if n > 0 {
		return errs.BadRequest(errs.ErrRoleInUse)
	}
	if err := s.UserRepo.DeleteRole(ctx, op.CompanyID, id); err != nil {
		return err
	}
	s.invalidateRoleAuth(ctx, op.CompanyID, id)
	return nil
}

// -------- 角色权限（RBAC）--------

// GetRolePerms 读取角色权限配置（权限配置界面回显）
func (s *Service) GetRolePerms(ctx context.Context, op Operator, roleID int64) (*dto.RolePermsResp, error) {
	role, err := s.UserRepo.GetRoleByID(ctx, op.CompanyID, roleID)
	if err != nil {
		return nil, errs.NotFound(errs.ErrRoleNotFound)
	}
	perms, err := s.UserRepo.ListPermsByRole(ctx, op.CompanyID, roleID)
	if err != nil {
		return nil, err
	}
	// 过滤代码中已删除的权限点，保证回显与判定口径一致
	valid := make([]string, 0, len(perms))
	for _, p := range perms {
		if domain.IsValidPerm(p) {
			valid = append(valid, p)
		}
	}
	return &dto.RolePermsResp{
		RoleID:      role.ID,
		RoleName:    role.Name,
		RoleCode:    role.Code,
		DataScope:   role.DataScope,
		Permissions: valid,
	}, nil
}

// GrantRolePerms 保存角色权限（全量覆盖：数据范围 + 权限点集合）。
//
// 保护规则：内置超管（admin）角色权限固定不可修改 —— 其权限由角色码短路获得，
// 即使改了仍会全量放行，允许修改只会造成"改了不生效"的困惑。
func (s *Service) GrantRolePerms(ctx context.Context, op Operator, roleID int64, req dto.RoleGrantReq) error {
	role, err := s.UserRepo.GetRoleByID(ctx, op.CompanyID, roleID)
	if err != nil {
		return errs.NotFound(errs.ErrRoleNotFound)
	}
	if domain.RoleCode(role.Code) == domain.RoleCodeAdmin {
		return errs.BadRequest(errs.ErrAdminRoleLocked)
	}
	scope := domain.DataScope(req.DataScope)
	if !scope.IsValid() {
		return errs.BadRequest(errs.ErrDataScopeInvalid)
	}
	// 逐项校验 + 去重，杜绝脏数据入库
	perms := make([]string, 0, len(req.Permissions))
	seen := make(map[string]struct{}, len(req.Permissions))
	for _, p := range req.Permissions {
		if !domain.IsValidPerm(p) {
			return errs.BadRequest(fmt.Sprintf("%s：%s", errs.ErrPermInvalid, p))
		}
		if _, dup := seen[p]; dup {
			continue
		}
		seen[p] = struct{}{}
		perms = append(perms, p)
	}
	// 数据范围与权限点同事务写入，避免"权限已生效但范围还是旧的"中间态
	if err := repository.Tx(func(tx *gorm.DB) error {
		repo := s.UserRepo.WithTx(tx)
		if err := repo.UpdateRole(ctx, op.CompanyID, roleID, map[string]interface{}{
			"data_scope": req.DataScope,
			"updated_by": op.UserID,
		}); err != nil {
			return err
		}
		return repo.ReplaceRolePerms(ctx, op.CompanyID, roleID, perms)
	}); err != nil {
		return err
	}
	s.invalidateRoleAuth(ctx, op.CompanyID, roleID)
	return nil
}

// PermCatalog 全量权限点分组（前端渲染权限配置树）
func (s *Service) PermCatalog() []domain.PermGroup {
	return domain.PermGroups()
}

// invalidateRoleAuth 清除角色授权缓存，使权限/数据范围变更即时生效
func (s *Service) invalidateRoleAuth(ctx context.Context, companyID, roleID int64) {
	if rdb := s.redis(); rdb != nil {
		authcache.DelRoleAuth(ctx, rdb, companyID, roleID)
	}
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
