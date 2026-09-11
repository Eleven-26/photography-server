package domain

// Perm 权限点标识，格式 resource:action（子动作用下划线，如 order:reschedule_audit）。
//
// 权限点作为**编译期常量**定义、不入库：IDE 可跳转、拼写错误在编译期暴露、不存在脏数据。
// 角色与权限的**绑定关系**存 sys_role_permission 表，租户可自定义角色权限。
//
// 新增权限点的流程：在此追加常量 → 在 permGroups 中归类 → 重启服务即生效。
// admin 角色通过角色码短路自动获得全部权限（含新增项，见 middleware/auth.go），
// 无需为其补数据。
type Perm string

const (
	// ---- 工作台 ----
	PermDashboardView Perm = "dashboard:view"

	// ---- 订单 ----
	// （order:price 已移除：订单金额无独立修改接口，改价经加项走 order:update；
	//   二期如落地改价接口再按 docs/rbac/运维手册.md 流程新增）
	PermOrderView            Perm = "order:view"
	PermOrderCreate          Perm = "order:create"
	PermOrderUpdate          Perm = "order:update"
	PermOrderStatus          Perm = "order:status"
	PermOrderCancel          Perm = "order:cancel"
	PermOrderReschedule      Perm = "order:reschedule"
	PermOrderRescheduleAudit Perm = "order:reschedule_audit"

	// ---- 客户 ----
	PermCustomerView   Perm = "customer:view"
	PermCustomerCreate Perm = "customer:create"
	PermCustomerUpdate Perm = "customer:update"
	PermCustomerDelete Perm = "customer:delete"

	// ---- 线索 ----
	PermLeadView    Perm = "lead:view"
	PermLeadCreate  Perm = "lead:create"
	PermLeadUpdate  Perm = "lead:update"
	PermLeadAssign  Perm = "lead:assign"
	PermLeadConvert Perm = "lead:convert"
	PermLeadDelete  Perm = "lead:delete"

	// ---- 报价 ----
	// （quote:audit 已移除：报价无审批接口，状态流转走 quote:update；
	//   二期如落地报价审批再新增）
	PermQuoteView   Perm = "quote:view"
	PermQuoteCreate Perm = "quote:create"
	PermQuoteUpdate Perm = "quote:update"

	// ---- 套餐 ----
	PermPackageView    Perm = "package:view"
	PermPackageCreate  Perm = "package:create"
	PermPackageUpdate  Perm = "package:update"
	PermPackagePublish Perm = "package:publish"
	PermPackageDelete  Perm = "package:delete"

	// ---- 收款 ----
	PermPaymentView    Perm = "payment:view"
	PermPaymentCreate  Perm = "payment:create"
	PermPaymentConfirm Perm = "payment:confirm"
	PermPaymentDelete  Perm = "payment:delete"

	// ---- 退款 ----
	PermRefundView   Perm = "refund:view"
	PermRefundCreate Perm = "refund:create"
	PermRefundAudit  Perm = "refund:audit"

	// ---- 交付 ----
	PermDeliveryView   Perm = "delivery:view"
	PermDeliveryCreate Perm = "delivery:create"
	PermDeliveryUpdate Perm = "delivery:update"
	PermDeliveryDelete Perm = "delivery:delete"

	// ---- 作品 ----
	PermAssetView   Perm = "asset:view"
	PermAssetUpload Perm = "asset:upload"
	PermAssetUpdate Perm = "asset:update"
	PermAssetAudit  Perm = "asset:audit"
	PermAssetDelete Perm = "asset:delete"

	// ---- 档期 ----
	PermCalendarView   Perm = "calendar:view"
	PermCalendarUpdate Perm = "calendar:update"

	// ---- 财务 ----
	PermFinanceView   Perm = "finance:view"
	PermFinanceExport Perm = "finance:export"

	// ---- 工作室设置 ----
	PermSettingsView   Perm = "settings:view"
	PermSettingsUpdate Perm = "settings:update"

	// ---- 成员（用户）----
	PermUserView     Perm = "user:view"
	PermUserCreate   Perm = "user:create"
	PermUserUpdate   Perm = "user:update"
	PermUserDelete   Perm = "user:delete"
	PermUserResetPwd Perm = "user:resetpwd"

	// ---- 角色 ----
	PermRoleView   Perm = "role:view"
	PermRoleCreate Perm = "role:create"
	PermRoleUpdate Perm = "role:update"
	PermRoleDelete Perm = "role:delete"
	PermRoleGrant  Perm = "role:grant"

	// ---- 门店 ----
	PermStoreView   Perm = "store:view"
	PermStoreCreate Perm = "store:create"
	PermStoreUpdate Perm = "store:update"
	PermStoreDelete Perm = "store:delete"

	// ---- 操作日志 ----
	PermLogView Perm = "log:view"

	// ---- 通知 ----
	PermNotificationView Perm = "notification:view"

	// ---- 定制需求 ----
	PermRequestView   Perm = "request:view"
	PermRequestHandle Perm = "request:handle"

	// ---- 评价 ----
	PermReviewView  Perm = "review:view"
	PermReviewReply Perm = "review:reply"

	// ---- 登录设备 ----
	PermDeviceView   Perm = "device:view"
	PermDeviceManage Perm = "device:manage"
)

// RoleCode 角色编码（对应 sys_role.code）。内置角色编码固定，用于代码内短路判定。
type RoleCode string

const (
	RoleCodeAdmin        RoleCode = "admin"        // 超级管理员：全部权限 + 全部数据
	RoleCodeManager      RoleCode = "manager"      // 店长
	RoleCodePhotographer RoleCode = "photographer" // 摄影师
	RoleCodeSales        RoleCode = "sales"        // 销售
)

// PermDesc 单个权限点的展示信息（供前端渲染配置界面）
type PermDesc struct {
	Key   Perm   `json:"key"`   // 权限点标识，如 order:view
	Label string `json:"label"` // 中文名，如 查看订单
}

// PermGroup 按业务模块分组的权限点集合（顺序即前端展示顺序）
type PermGroup struct {
	Module string     `json:"module"` // 模块中文名
	Perms  []PermDesc `json:"perms"`
}

// permGroups 全量权限点分组定义 —— 新增权限点在此归类，勿遗漏。
var permGroups = []PermGroup{
	{Module: "工作台", Perms: []PermDesc{
		{PermDashboardView, "查看工作台"},
	}},
	{Module: "订单", Perms: []PermDesc{
		{PermOrderView, "查看订单"},
		{PermOrderCreate, "创建订单"},
		{PermOrderUpdate, "编辑订单"},
		{PermOrderStatus, "变更订单状态"},
		{PermOrderCancel, "取消订单"},
		{PermOrderReschedule, "申请改期"},
		{PermOrderRescheduleAudit, "审批改期"},
	}},
	{Module: "客户", Perms: []PermDesc{
		{PermCustomerView, "查看客户"},
		{PermCustomerCreate, "创建客户"},
		{PermCustomerUpdate, "编辑客户"},
		{PermCustomerDelete, "删除客户"},
	}},
	{Module: "线索", Perms: []PermDesc{
		{PermLeadView, "查看线索"},
		{PermLeadCreate, "创建线索"},
		{PermLeadUpdate, "编辑 / 跟进线索"},
		{PermLeadAssign, "分配线索"},
		{PermLeadConvert, "线索转客户"},
		{PermLeadDelete, "删除线索"},
	}},
	{Module: "报价", Perms: []PermDesc{
		{PermQuoteView, "查看报价"},
		{PermQuoteCreate, "创建报价"},
		{PermQuoteUpdate, "编辑报价"},
	}},
	{Module: "套餐", Perms: []PermDesc{
		{PermPackageView, "查看套餐"},
		{PermPackageCreate, "创建套餐"},
		{PermPackageUpdate, "编辑套餐"},
		{PermPackagePublish, "上下架套餐"},
		{PermPackageDelete, "删除套餐"},
	}},
	{Module: "收款", Perms: []PermDesc{
		{PermPaymentView, "查看收款"},
		{PermPaymentCreate, "登记收款"},
		{PermPaymentConfirm, "核验到账"},
		{PermPaymentDelete, "删除收款记录"},
	}},
	{Module: "退款", Perms: []PermDesc{
		{PermRefundView, "查看退款"},
		{PermRefundCreate, "发起退款"},
		{PermRefundAudit, "审批退款"},
	}},
	{Module: "交付", Perms: []PermDesc{
		{PermDeliveryView, "查看交付"},
		{PermDeliveryCreate, "创建交付任务"},
		{PermDeliveryUpdate, "处理交付"},
		{PermDeliveryDelete, "删除交付"},
	}},
	{Module: "作品", Perms: []PermDesc{
		{PermAssetView, "查看作品"},
		{PermAssetUpload, "上传作品"},
		{PermAssetUpdate, "编辑作品"},
		{PermAssetAudit, "审核作品"},
		{PermAssetDelete, "删除作品"},
	}},
	{Module: "档期", Perms: []PermDesc{
		{PermCalendarView, "查看档期"},
		{PermCalendarUpdate, "维护档期"},
	}},
	{Module: "财务", Perms: []PermDesc{
		{PermFinanceView, "查看财务报表"},
		{PermFinanceExport, "导出财务数据"},
	}},
	{Module: "工作室设置", Perms: []PermDesc{
		{PermSettingsView, "查看工作室设置"},
		{PermSettingsUpdate, "修改工作室设置"},
	}},
	{Module: "成员管理", Perms: []PermDesc{
		{PermUserView, "查看成员"},
		{PermUserCreate, "新增成员"},
		{PermUserUpdate, "编辑成员"},
		{PermUserDelete, "删除成员"},
		{PermUserResetPwd, "重置成员密码"},
	}},
	{Module: "角色权限", Perms: []PermDesc{
		{PermRoleView, "查看角色"},
		{PermRoleCreate, "新增角色"},
		{PermRoleUpdate, "编辑角色"},
		{PermRoleDelete, "删除角色"},
		{PermRoleGrant, "配置角色权限"},
	}},
	{Module: "门店管理", Perms: []PermDesc{
		{PermStoreView, "查看门店"},
		{PermStoreCreate, "新增门店"},
		{PermStoreUpdate, "编辑门店"},
		{PermStoreDelete, "删除门店"},
	}},
	{Module: "操作日志", Perms: []PermDesc{
		{PermLogView, "查看操作日志"},
	}},
	{Module: "通知管理", Perms: []PermDesc{
		{PermNotificationView, "管理通知"},
	}},
	{Module: "定制需求", Perms: []PermDesc{
		{PermRequestView, "查看定制需求"},
		{PermRequestHandle, "处理定制需求"},
	}},
	{Module: "客户评价", Perms: []PermDesc{
		{PermReviewView, "查看评价"},
		{PermReviewReply, "回复评价"},
	}},
	{Module: "登录设备", Perms: []PermDesc{
		{PermDeviceView, "查看登录设备"},
		{PermDeviceManage, "管理登录设备"},
	}},
}

var (
	// allPerms 全量权限点（扁平，顺序与 permGroups 一致）
	allPerms []Perm
	// permSet 权限点集合，用于 O(1) 合法性校验
	permSet map[string]struct{}
)

func init() {
	for _, g := range permGroups {
		for _, p := range g.Perms {
			allPerms = append(allPerms, p.Key)
		}
	}
	permSet = make(map[string]struct{}, len(allPerms))
	for _, p := range allPerms {
		permSet[string(p)] = struct{}{}
	}
}

// PermGroups 返回全量权限点分组（供前端渲染权限配置界面）
func PermGroups() []PermGroup { return permGroups }

// AllPermissions 返回全量权限点扁平列表
func AllPermissions() []Perm { return allPerms }

// IsValidPerm 校验权限点是否合法（入库前校验，杜绝脏数据）
func IsValidPerm(p string) bool {
	_, ok := permSet[p]
	return ok
}

// HasPerm 判断权限集合是否包含指定权限点
func HasPerm(perms []string, want Perm) bool {
	for _, p := range perms {
		if p == string(want) {
			return true
		}
	}
	return false
}

// HasAnyPerm 判断权限集合是否包含任一指定权限点（空需求列表视为通过）
func HasAnyPerm(perms []string, wants ...Perm) bool {
	if len(wants) == 0 {
		return true
	}
	for _, w := range wants {
		if HasPerm(perms, w) {
			return true
		}
	}
	return false
}
