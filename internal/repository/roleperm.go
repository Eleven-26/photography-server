package repository

import (
	"context"
	"time"

	"photography-server/internal/model"
)

// -------- 角色权限（RBAC）--------
//
// 方法挂在 UserRepo 上：角色与成员同属「用户与权限」域，且需复用其 WithTx 事务透传。
// 权限点本身不在库中（见 domain/perm.go），本表只存「角色 → 权限点」的绑定关系。

// ListPermsByRole 读取某角色的权限点集合（按权限点升序）
func (r *UserRepo) ListPermsByRole(ctx context.Context, companyID, roleID int64) ([]string, error) {
	perms := make([]string, 0, 16)
	err := r.tenant(companyID).WithContext(ctx).
		Model(&model.SysRolePermission{}).
		Where("role_id = ?", roleID).
		Order("permission ASC").
		Pluck("permission", &perms).Error
	return perms, err
}

// CountPermsByRoles 统计各角色的权限点数量，返回 roleID → 数量。
// 供角色列表展示「已配置 N 项权限」，避免 N+1 查询。
func (r *UserRepo) CountPermsByRoles(ctx context.Context, companyID int64) (map[int64]int, error) {
	var rows []struct {
		RoleID int64
		N      int64
	}
	err := r.tenant(companyID).WithContext(ctx).
		Model(&model.SysRolePermission{}).
		Select("role_id, COUNT(*) AS n").
		Group("role_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	m := make(map[int64]int, len(rows))
	for _, row := range rows {
		m[row.RoleID] = int(row.N)
	}
	return m, nil
}

// ReplaceRolePerms 全量覆盖某角色的权限（先物理删除旧记录，再批量插入）。
//
// 采用物理删除而非软删：uk_role_perm(company_id,role_id,permission) 是唯一键，
// 软删后重新添加同一权限会命中旧记录导致唯一键冲突；本表无追溯价值，物理删更简单。
//
// 必须在外层事务内调用（repository.Tx），否则删除与插入之间存在空窗期，
// 期间并发请求会读到"角色无任何权限"的中间态。
func (r *UserRepo) ReplaceRolePerms(ctx context.Context, companyID, roleID int64, perms []string) error {
	db := r.conn().WithContext(ctx)
	if err := db.Where("company_id = ? AND role_id = ?", companyID, roleID).
		Delete(&model.SysRolePermission{}).Error; err != nil {
		return err
	}
	if len(perms) == 0 {
		return nil
	}
	now := time.Now()
	rows := make([]model.SysRolePermission, 0, len(perms))
	for _, p := range perms {
		rows = append(rows, model.SysRolePermission{
			CompanyID:  companyID,
			RoleID:     roleID,
			Permission: p,
			CreatedAt:  now,
		})
	}
	return db.Create(&rows).Error
}

// GetRoleByID 按 ID 取角色（自动带 company_id 租户过滤）
func (r *UserRepo) GetRoleByID(ctx context.Context, companyID, roleID int64) (*model.SysRole, error) {
	var role model.SysRole
	if err := r.tenant(companyID).WithContext(ctx).First(&role, roleID).Error; err != nil {
		return nil, err
	}
	return &role, nil
}

// CountUsersByRole 统计使用某角色的成员数（删除角色前的占用校验）
func (r *UserRepo) CountUsersByRole(ctx context.Context, companyID, roleID int64) (int64, error) {
	var n int64
	err := r.tenant(companyID).WithContext(ctx).
		Model(&model.SysUser{}).
		Where("role_id = ?", roleID).
		Count(&n).Error
	return n, err
}
