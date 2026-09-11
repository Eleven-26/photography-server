package repository

import (
	"gorm.io/gorm"

	"photography-server/internal/domain"
)

// ScopeCols 声明某张表参与行级数据权限过滤的列。
//
// 由各仓储方法**显式声明**，而非依赖自动推断——表结构差异大（有的按 store_id，
// 有的按 photographer_id/owner_id/created_by），自动推断会猜错并造成越权。
type ScopeCols struct {
	// Store 门店过滤列名（如 "store_id"）；空串表示本表无门店维度
	Store string
	// Owner 归属人过滤列名（多列之间取 OR，如 {"photographer_id","owner_id"}）；
	// 空切片表示本表无归属人维度
	Owner []string
}

// applyScope 按操作人的数据范围给查询追加行级过滤条件。
//
// 语义：
//   - ScopeAll（含 DataScope 零值）→ 不加任何条件。零值放行是有意为之：
//     权限体系未启用 / 存量角色未配置数据范围时，行为与改造前完全一致。
//   - ScopeStore → `<Store> = op.StoreID`
//   - ScopeSelf  → `<Owner[0]> = op.UserID OR <Owner[1]> = op.UserID ...`
//
// 降级规则（安全优先，绝不因"维度缺失"而放行全部数据）：
//   - op.StoreID == 0（未分配门店）而需要门店过滤 → 降级为 ScopeSelf，
//     否则 `store_id = 0` 会查到全部「未分配门店」的数据，属越权；
//   - 表未声明 Owner 列却需要 Self 过滤 → 返回 `1 = 0`（查不到数据）。
//     确需放行的共享资源（如套餐为全店共享）应由调用方决定**不调用**本函数。
func applyScope(q *gorm.DB, op domain.Operator, c ScopeCols) *gorm.DB {
	switch op.DataScope {
	case domain.ScopeStore:
		if c.Store != "" && op.StoreID != 0 {
			return q.Where(c.Store+" = ?", op.StoreID)
		}
		return applySelf(q, op, c)
	case domain.ScopeSelf:
		return applySelf(q, op, c)
	}
	return q
}

// applySelf 追加「归属人为本人」的过滤（多列 OR）
func applySelf(q *gorm.DB, op domain.Operator, c ScopeCols) *gorm.DB {
	if len(c.Owner) == 0 {
		return q.Where("1 = 0")
	}
	cond := ""
	args := make([]interface{}, 0, len(c.Owner))
	for i, col := range c.Owner {
		if i > 0 {
			cond += " OR "
		}
		cond += col + " = ?"
		args = append(args, op.UserID)
	}
	return q.Where(cond, args...)
}
