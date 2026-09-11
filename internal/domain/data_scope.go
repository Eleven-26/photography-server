package domain

// DataScope 数据范围（行级数据权限），绑定在角色上（sys_role.data_scope）。
//
// 语义：
//
//	ScopeAll   全部数据 —— 不加任何过滤条件（默认值，保证存量角色行为不变）
//	ScopeStore 本门店   —— 按数据行的 store_id 过滤
//	ScopeSelf  仅本人   —— 按归属人过滤
//	                      订单 = photographer_id 或 owner_id；线索 = owner_id；
//	                      客户 = created_by；档期 = photographer_id
//
// 特殊规则：
//   - Operator.StoreID == 0（未分配门店）时 ScopeStore 降级为 ScopeSelf，
//     否则 `store_id = 0` 会查到全部"未分配门店"的数据，属越权；
//   - admin 角色恒为 ScopeAll，防止管理员把自己锁死。
//
// 注入方式见 repository/scope.go 的 applyScope：仓储方法显式声明本表的作用列，
// 无门店维度的表（如 biz_asset 已补列；biz_delivery/payment/refund 等子表）
// 需 JOIN biz_order 后按主表列过滤，不能依赖自动注入。
type DataScope int

const (
	ScopeAll   DataScope = 1 // 全部数据
	ScopeStore DataScope = 2 // 本门店
	ScopeSelf  DataScope = 3 // 仅本人
)

// IsValid 校验数据范围取值合法性（接口入参 / 入库前校验）
func (d DataScope) IsValid() bool {
	switch d {
	case ScopeAll, ScopeStore, ScopeSelf:
		return true
	}
	return false
}

// String 中文描述（错误消息 / 界面展示）
func (d DataScope) String() string {
	switch d {
	case ScopeAll:
		return "全部数据"
	case ScopeStore:
		return "本门店"
	case ScopeSelf:
		return "仅本人"
	}
	return "未知"
}
