package repository

import (
	"context"
	"strings"

	"gorm.io/gorm"

	"photography-server/internal/domain"
	"photography-server/internal/model"
)

// ScopeCols 声明某张表参与行级数据权限过滤的列。
//
// 由各仓储方法**显式声明**，而非依赖自动推断——表结构差异大（有的按 store_id，
// 有的按 photographer_id/owner_id/created_by），自动推断会猜错并造成越权。
type ScopeCols struct {
	// Store 门店过滤列名（如 "store_id"；JOIN 场景带表别名如 "o.store_id"）；
	// 空串表示本表无门店维度
	Store string
	// Owner 归属人过滤列名（多列之间取 OR，如 {"photographer_id","owner_id"}）；
	// 空切片表示本表无归属人维度
	Owner []string
	// Shared 标记该表是「门店级共享资源」（如套餐）：ScopeSelf 时按 Store 列过滤
	// （本人也看得见本店资源），而非按 Owner 列过滤。
	// 不置此位时，ScopeSelf 会因本表无 Owner 列而落进 `1 = 0`，把共享资源全部锁死。
	Shared bool
	// Public 标记该表存在「公共池」：Store 列为 0 的行属无主数据（如 H5 游客提交的
	// 定制需求，提交时尚无门店归属），对任何员工可见，由员工响应认领。
	// 生效于 ScopeStore 与 ScopeSelf：过滤条件会附加 `OR <Store> = 0`。
	Public bool
}

// 各业务表的过滤列声明，集中定义避免列名字符串散落在各仓储方法内。
var (
	scopeOrder    = ScopeCols{Store: "store_id", Owner: []string{"photographer_id", "owner_id"}}
	scopeCustomer = ScopeCols{Store: "store_id", Owner: []string{"created_by"}}
	scopeLead     = ScopeCols{Store: "store_id", Owner: []string{"owner_id"}}
	scopeCalendar = ScopeCols{Store: "store_id", Owner: []string{"photographer_id"}}
	scopeAsset    = ScopeCols{Store: "store_id", Owner: []string{"created_by"}}
	// 套餐是门店级共享资源：摄影师（仅本人）也必须看得到本店套餐，否则无法开单。
	scopePackage = ScopeCols{Store: "store_id", Shared: true}

	// 定制需求：独立接单 + 公共池。H5 客户提交时尚无门店归属（store_id=0），
	// 本店需能看到无主需求去认领，故 Public；已响应的记录归响应人（response_by）。
	scopeCustomRequest = ScopeCols{Store: "store_id", Owner: []string{"response_by"}, Public: true}

	// 已 JOIN biz_order AS o 的场景（交片单及其明细）直接按订单归属过滤
	scopeOrderJoined = ScopeCols{Store: "o.store_id", Owner: []string{"o.photographer_id", "o.owner_id"}}
)

// opOf 取当前操作人。链路未经员工认证（客户侧接口 / 定时任务 / 单测）时返回零值
// Operator —— 零值 DataScope（0）在 applyScope 中走「不加条件」分支，
// 因此这类链路行为与改造前完全一致。
func opOf(ctx context.Context) domain.Operator {
	op, _ := domain.OperatorFrom(ctx)
	return op
}

// scopedOrder 给订单表查询追加数据权限（最常用，单独抽出避免重复传参）。
func scopedOrder(q *gorm.DB, ctx context.Context) *gorm.DB {
	return applyScope(q, opOf(ctx), scopeOrder)
}

// visibleOrderIDs 返回「当前操作人可见订单 ID」的子查询，供**无 store_id 的从表**
// （biz_order_payment / biz_order_refund / biz_order_addon / biz_order_reschedule /
// biz_delivery）过滤：`q.Where("order_id IN (?)", r.visibleOrderIDs(ctx, companyID))`。
//
// 用子查询而非再 JOIN 一次订单表，是为了避免 JOIN 下方列名歧义与分页 COUNT 重复行。
func (r *Repo) visibleOrderIDs(ctx context.Context, companyID int64) *gorm.DB {
	return applyScope(r.tenant(companyID).Model(&model.Order{}).Select("id"), opOf(ctx), scopeOrder)
}

// scopedFromOrder 给**无 store_id 的订单从表**查询追加数据权限（经订单可见性传递）。
//
// 仅在确实需要过滤（本门店 / 仅本人）时追加子查询；ScopeAll 与未认证链路原样返回，
// 既保证行为不变，也避免每次查询都白跑一个 `order_id IN (SELECT ...)`。
func (r *Repo) scopedFromOrder(q *gorm.DB, ctx context.Context, companyID int64) *gorm.DB {
	op := opOf(ctx)
	if op.DataScope != domain.ScopeStore && op.DataScope != domain.ScopeSelf {
		return q
	}
	return q.Where("order_id IN (?)", r.visibleOrderIDs(ctx, companyID))
}

// applyScope 按操作人的数据范围给查询追加行级过滤条件。
//
// 语义：
//   - ScopeAll（含 DataScope 零值）→ 不加任何条件。零值放行是有意为之：
//     权限体系未启用 / 存量角色未配置数据范围 / 未经员工认证的链路时，
//     行为与改造前完全一致。
//   - ScopeStore → `<Store> = op.StoreID`（Public 表附加 `OR <Store> = 0` 公共池）
//   - ScopeSelf  → `<Owner[0]> = op.UserID OR <Owner[1]> = op.UserID ...`
//     （Shared 表降级为按 Store 过滤，见 ScopeCols.Shared；
//     Public 表附加 `OR <Store> = 0` 公共池，保证无主数据可被认领）
//
// 降级规则（安全优先，绝不因"维度缺失"而放行全部数据）：
//   - op.StoreID == 0（未分配门店）而需要门店过滤 → 降级为 ScopeSelf，
//     否则 `store_id = 0` 会查到全部「未分配门店」的数据，属越权；
//   - 表未声明 Owner 列却需要 Self 过滤 → 返回 `1 = 0`（查不到数据）。
//     确需放行的共享资源应由调用方在 ScopeCols 上标记 Shared / Public 或**不调用**本函数。
func applyScope(q *gorm.DB, op domain.Operator, c ScopeCols) *gorm.DB {
	switch op.DataScope {
	case domain.ScopeStore:
		if c.Store != "" && op.StoreID != 0 {
			return q.Where(storeCond(c), op.StoreID)
		}
		return applySelf(q, op, c)
	case domain.ScopeSelf:
		if c.Shared && c.Store != "" && op.StoreID != 0 {
			return q.Where(storeCond(c), op.StoreID)
		}
		return applySelf(q, op, c)
	}
	return q
}

// storeCond 生成门店维度过滤条件；Public 表的 store_id=0 视为公共池对本店可见
func storeCond(c ScopeCols) string {
	if c.Public {
		return "(" + c.Store + " = ? OR " + c.Store + " = 0)"
	}
	return c.Store + " = ?"
}

// applySelf 追加「归属人为本人」的过滤（多列 OR）
func applySelf(q *gorm.DB, op domain.Operator, c ScopeCols) *gorm.DB {
	conds := make([]string, 0, len(c.Owner)+1)
	args := make([]interface{}, 0, len(c.Owner)+1)
	for _, col := range c.Owner {
		conds = append(conds, col+" = ?")
		args = append(args, op.UserID)
	}
	if c.Public && c.Store != "" {
		conds = append(conds, c.Store+" = 0")
	}
	if len(conds) == 0 {
		return q.Where("1 = 0")
	}
	return q.Where(strings.Join(conds, " OR "), args...)
}
