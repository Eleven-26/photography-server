package router

import (
	"reflect"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"photography-server/internal/presentation/endpoint"
	"photography-server/internal/presentation/h5"
	"photography-server/internal/presentation/routes"
)

// 端无关路由表的架构护栏（2026-09-12 路由整理，对应方案 4.3）。
//
// 这是本次整理**最高性价比的产出**：即使将来有人图快又抄一遍 handler、
// 或新增路由漏挂权限点、或员工端悄悄多挂/漏挂，测试会直接红。
//
// 测试无需 DB / 配置：所有中间件都是「返回闭包」形态（如 mw.Perm 返回 func(c)），
// 对 nil 指针取方法值、对 nil 接收者构造闭包都不会真正执行，因此可直接枚举真实路由。

// permExempt 允许免挂权限点的路径白名单（自助类 / 通用能力）。
// 新增免挂路由必须在此登记，否则 TestPermCoverage 失败。
var permExempt = map[string]bool{
	"/user/profile":         true, // 操作对象是登录者本人账号
	"/user/change-password": true,
	"/user/logout":          true,
	"/upload/file":          true, // 通用上传能力，被收款凭证/作品/交付等多条链路共用
	"/device/list":          true, // 员工端：本人登录设备
	"/device/remove/:id":    true,
	"/user/mobile-code":     true, // 员工端：换绑手机号（操作对象是登录者本人）
	"/user/change-mobile":   true,
	"/feedback/submit":      true, // 员工端：意见反馈（提交人本人）
	// 员工端登录出口（见 endpoints.go 的 staffPublicExtra）：登录发生在拿到令牌之前，
	// 天然不能挂权限点，归入 PublicExtra 免端级中间件。
	// 路径与客户区同名（/auth/login 等）但前缀不同——客户区不参与本表扫描
	// （其鉴权由 CustomerAuth 分组承担，见 TestClientRouteTableIntegrity 的 Perm 断言）。
	"/auth/sms-code":      true,
	"/auth/login":         true,
	"/auth/login-by-code": true,
}

// staffRouteCountBaseline 员工端路由数基线：
// 52（与 PC 同路径复用，含补开的 4 条 /notification/*、2 条取消/改期发起、1 条 /delivery/items/:id，
// 2026-09-14 补开的 8 条：/customer/create、账号自助 /user/{profile,change-password,logout}、
// 收款方式 /settings/payment-method/{list,create,update,delete}，
// 以及同日第四批补开的 15 条：套餐 /package/* 6 条、作品集 /asset/* 6 条、报价 /quote/* 3 条，
// 同日第五批补开的 2 条：/delivery/select/:id（选片结果页）、/customer/orders/:id（客户档案订单列表））
// + 6（异路径别名）+ 2（真实端差异）+ 18（移动端独有）= 80。
//
// 2026-09-14 第六批（B 类页面补建）**全部走 Extra**，Include 不变：
//
//	/delivery/send-final/:id、/user/mobile-code、/user/change-mobile、/feedback/submit → +4 = 84。
//
// 2026-09-14 第七批：定制需求 /custom-request/{list,respond/:id} 自 Extra **上提到 Include**，
// 与 PC 管理端共用同一 Handler（消除同路径两端各自实现的契约漂移）→ Include +2 / Extra -2，总数仍 84。
//
// 2026-09-16：三条登录出口（/auth/{sms-code,login,login-by-code}）自 presentation/wechat/staff.go
// 的内联 g.POST 迁到 staffPublicExtra（免 StaffAuth 的 PublicExtra），首次纳入本基线统计 → +3 = 87。
// **统计口径 = Include + Extra + PublicExtra**，即整端暴露面。
//
// 有意增减员工端暴露面时须同步更新此基线——它防的是"悄悄多挂 / 漏挂"。
const staffRouteCountBaseline = 87

// clientPublicCountBaseline / clientAuthedCountBaseline 客户区路由数基线。
// 客户区路由由 H5 与小程序**共用同一份声明**（routes.ClientPublic / ClientAuthed），
// 故只有一份计数——两端各自挂载时集合必然一致，无需按端分别统计。
const clientPublicCountBaseline = 9
const clientAuthedCountBaseline = 39

// commonRouteCountBaseline 管理端业务路由数基线（PC 与小程序共用）。
// 2026-09-14：新增定制需求 3 条（list / respond / convert）→ 111。
const commonRouteCountBaseline = 111

// staffAliases 员工端异路径别名 → 公共路由表对应路径。
// 别名路径的 Handler 必须与公共实现是同一函数，否则就是又抄了一遍。
var staffAliases = map[string]string{
	"/slot-template/list":       "/calendar/slot-template/list",
	"/slot-template/save":       "/calendar/slot-template/save",
	"/slot-template/save/:id":   "/calendar/slot-template/save/:id",
	"/slot-template/delete/:id": "/calendar/slot-template/delete/:id",
	"/studio/get":               "/settings/studio/get",
	"/studio/update":            "/settings/studio/update",
}

// mountForTest 用零依赖（nil ctl / nil mw）构造引擎并返回 "METHOD /path" → Handler 函数指针。
func mountForTest(ep endpoint.Endpoint, common []routes.Route) map[string]uintptr {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	endpoint.Mount(engine.Group(""), ep, common, nil)
	out := map[string]uintptr{}
	for _, ri := range engine.Routes() {
		out[ri.Method+" "+ri.Path] = reflect.ValueOf(ri.Handler).Pointer()
	}
	return out
}

// TestCommonRouteTableIntegrity 公共路由表自身完整性：路径合法、Handler 非空、无重复。
func TestCommonRouteTableIntegrity(t *testing.T) {
	common := routes.Common(nil)
	if len(common) != commonRouteCountBaseline {
		t.Errorf("公共路由数 %d，期望 %d（有意的增减请更新基线）", len(common), commonRouteCountBaseline)
	}
	seen := map[string]bool{}
	for _, r := range common {
		if r.Path == "" || !strings.HasPrefix(r.Path, "/") {
			t.Errorf("非法路径: %q", r.Path)
		}
		if r.Handler == nil {
			t.Errorf("%s 的 Handler 为空", r.Path)
		}
		if seen[r.Path] {
			t.Errorf("公共路由表路径重复（会导致 gin 注册冲突）: %s", r.Path)
		}
		seen[r.Path] = true
	}
}

// TestPermCoverage 权限点覆盖：除豁免白名单外，每条路由都必须挂权限点。
func TestPermCoverage(t *testing.T) {
	for _, r := range routes.Common(nil) {
		if r.Perm == "" && !permExempt[r.Path] {
			t.Errorf("[pc/miniapp] 路由 %s 未挂权限点且不在豁免白名单", r.Path)
		}
	}
	for _, r := range staffExtra(nil, nil) {
		if r.Perm == "" && !permExempt[r.Path] {
			t.Errorf("[staff] 路由 %s 未挂权限点且不在豁免白名单", r.Path)
		}
	}
	for _, r := range staffPublicExtra(nil) {
		if r.Perm == "" && !permExempt[r.Path] {
			t.Errorf("[staff] 登录出口 %s 未挂权限点且不在豁免白名单", r.Path)
		}
	}
}

// TestStaffDeclarationIntegrity 员工端声明完整性：
// Include 白名单必须存在于公共路由表；不得重复注册；路由数符合基线；
// 异路径别名的 Handler 必须与公共实现是同一函数。
func TestStaffDeclarationIntegrity(t *testing.T) {
	common := routes.Common(nil)
	byPath := map[string]routes.Route{}
	for _, r := range common {
		byPath[r.Path] = r
	}

	// 1) Include 白名单必须可解析
	for _, p := range staffInclude {
		if _, ok := byPath[p]; !ok {
			t.Errorf("staffInclude 引用了公共路由表不存在的路径: %s", p)
		}
	}

	// 2) 不得重复注册
	seen := map[string]bool{}
	for _, p := range staffInclude {
		if seen[p] {
			t.Errorf("staffInclude 内部重复: %s", p)
		}
		seen[p] = true
	}
	extra := staffExtra(nil, nil)
	public := staffPublicExtra(nil)
	for _, r := range extra {
		if seen[r.Path] {
			t.Errorf("员工端路由重复注册（会导致 gin 冲突）: %s", r.Path)
		}
		seen[r.Path] = true
	}
	for _, r := range public {
		if seen[r.Path] {
			t.Errorf("员工端登录出口与其它路由重复（会导致 gin 冲突）: %s", r.Path)
		}
		seen[r.Path] = true
	}

	// 3) 路由数基线（Include + Extra + PublicExtra = 整端暴露面）
	if got := len(staffInclude) + len(extra) + len(public); got != staffRouteCountBaseline {
		t.Errorf("员工端路由数 %d，期望 %d（有意的增减请更新基线）", got, staffRouteCountBaseline)
	}

	// 4) 异路径别名必须复用同一 Handler 函数指针
	for _, r := range extra {
		commonPath, isAlias := staffAliases[r.Path]
		if !isAlias {
			continue
		}
		want := byPath[commonPath].Handler
		if reflect.ValueOf(r.Handler).Pointer() != reflect.ValueOf(want).Pointer() {
			t.Errorf("员工端 %s 的 Handler 与公共实现 %s 不是同一函数（重复实现）", r.Path, commonPath)
		}
	}
}

// TestCrossEndpointHandlerIdentity 跨端同一性：员工端复用的每条路径，
// handler 必须与 PC 端是**同一函数指针**——这是"不再复制 handler"的硬保证。
func TestCrossEndpointHandlerIdentity(t *testing.T) {
	common := routes.Common(nil)
	pc := mountForTest(pcEndpoint(nil), common)
	staff := mountForTest(staffEndpoint(nil, nil, nil), common)

	for _, p := range staffInclude {
		pcKey := "POST " + p
		stKey := "POST /wechat/staff" + p
		pcPtr, okPC := pc[pcKey]
		stPtr, okStaff := staff[stKey]
		if !okPC {
			t.Errorf("PC 端未注册 %s", pcKey)
			continue
		}
		if !okStaff {
			t.Errorf("员工端未注册 %s", stKey)
			continue
		}
		if pcPtr != stPtr {
			t.Errorf("%s 跨端 Handler 不同：员工端没有复用 PC 实现", p)
		}
	}
}

// TestMiniappMirrorsPC 小程序管理端与 PC 同源：路由集合与 Handler 必须完全一致（仅前缀不同）。
func TestMiniappMirrorsPC(t *testing.T) {
	common := routes.Common(nil)
	pc := mountForTest(pcEndpoint(nil), common)
	ma := mountForTest(miniappEndpoint(nil), common)

	if len(pc) != len(ma) {
		t.Fatalf("小程序端路由数 %d != PC %d", len(ma), len(pc))
	}
	for key, pcPtr := range pc {
		maKey := "POST /miniapp" + strings.TrimPrefix(key, "POST ")
		maPtr, ok := ma[maKey]
		if !ok {
			t.Errorf("小程序端缺少 %s", maKey)
			continue
		}
		if maPtr != pcPtr {
			t.Errorf("%s 小程序端与 PC 的 Handler 不同", key)
		}
	}
}

// TestPCContainsAllCommonRoutes PC 端必须注册公共路由表的每一条（无漏挂）。
func TestPCContainsAllCommonRoutes(t *testing.T) {
	common := routes.Common(nil)
	pc := mountForTest(pcEndpoint(nil), common)
	if len(pc) != len(common) {
		t.Fatalf("PC 注册路由数 %d != 公共路由表 %d", len(pc), len(common))
	}
	for _, r := range common {
		if _, ok := pc["POST "+r.Path]; !ok {
			t.Errorf("PC 端漏挂 %s", r.Path)
		}
	}
}

// ---------------------------------------------------------------------
// 客户区（H5 与小程序客户区共用同一份声明）
// ---------------------------------------------------------------------

// TestClientRouteTableIntegrity 客户区路由表完整性。
//
// 客户区与另两套声明（管理端 Common / 员工端 staffInclude+Extra）的**关键差异**：
// 它不使用 RBAC 权限点——能否访问由挂载分组决定（免鉴权组 / CustomerAuth 组）。
// 故此处断言 Perm 必须为空：若有人把员工权限点误挂到客户区，PC 的角色配置会莫名
// 影响客户侧接口，而权限护栏（TestPermCoverage）只扫管理端与员工端，不会报错。
func TestClientRouteTableIntegrity(t *testing.T) {
	pub := routes.ClientPublic(nil)
	authed := routes.ClientAuthed(nil)

	if len(pub) != clientPublicCountBaseline {
		t.Errorf("客户区公开路由数 %d，期望 %d（有意的增减请更新基线）", len(pub), clientPublicCountBaseline)
	}
	if len(authed) != clientAuthedCountBaseline {
		t.Errorf("客户区需登录路由数 %d，期望 %d（有意的增减请更新基线）", len(authed), clientAuthedCountBaseline)
	}

	seen := map[string]bool{}
	groups := []struct {
		name string
		rs   []routes.Route
	}{{"public", pub}, {"authed", authed}}
	for _, g := range groups {
		for _, r := range g.rs {
			if r.Path == "" || !strings.HasPrefix(r.Path, "/") {
				t.Errorf("[%s] 非法路径: %q", g.name, r.Path)
			}
			if r.Handler == nil {
				t.Errorf("[%s] %s 的 Handler 为空", g.name, r.Path)
			}
			if r.Perm != "" {
				t.Errorf("[%s] %s 挂了权限点 %q —— 客户区鉴权由分组中间件（CustomerAuth）承担，"+
					"不应使用员工 RBAC 权限点", g.name, r.Path, r.Perm)
			}
			if seen[r.Path] {
				t.Errorf("客户区路径重复（会导致 gin 注册冲突）: %s", r.Path)
			}
			seen[r.Path] = true
		}
	}
}

// TestClientEndpointsShareHandler H5 与小程序客户区必须挂同一份路由表。
//
// 复用由构造保证：两端各自调用 routes.ClientPublic/ClientAuthed，拿到的是同一份声明，
// 只是挂在不同前缀。本测试模拟 router.go 的两次挂载，断言"同一条相对路径在两端解析到
// 同一个 Handler 函数指针"——若将来有人给小程序单独建一张表（回到"复制一份"的老路），
// 这里会红。此前靠 wechat 包转调 h5 的注册函数达成复用，但客户区路由游离在声明之外，
// 护栏测不到；改为声明式后这条保证才真正可断言。
func TestClientEndpointsShareHandler(t *testing.T) {
	ctl := h5.New(nil, nil)
	pub := routes.ClientPublic(ctl)
	authed := routes.ClientAuthed(ctl)

	mount := func(prefix string) map[string]uintptr {
		gin.SetMode(gin.TestMode)
		engine := gin.New()
		g := engine.Group(prefix)
		endpoint.MountTable(g, pub)
		endpoint.MountTable(g, authed)
		out := map[string]uintptr{}
		for _, ri := range engine.Routes() {
			out[strings.TrimPrefix(ri.Path, prefix)] = reflect.ValueOf(ri.Handler).Pointer()
		}
		return out
	}

	h5Routes := mount("/h5")
	wxRoutes := mount("/wechat")

	want := clientPublicCountBaseline + clientAuthedCountBaseline
	if len(h5Routes) != want {
		t.Fatalf("H5 客户区挂载路由数 %d，期望 %d", len(h5Routes), want)
	}
	if len(wxRoutes) != len(h5Routes) {
		t.Fatalf("小程序客户区路由数 %d != H5 %d", len(wxRoutes), len(h5Routes))
	}
	for p, ptr := range h5Routes {
		wxPtr, ok := wxRoutes[p]
		if !ok {
			t.Errorf("小程序客户区缺少 %s", p)
			continue
		}
		if wxPtr != ptr {
			t.Errorf("客户区 %s 两端 Handler 不同：小程序没有复用 H5 实现", p)
		}
	}
}

// TestStaffPublicExtraNotGuarded 员工端登录出口不得挂认证中间件保护的权限点，
// 且必须能在**免 StaffAuth** 的分组下注册成功——这是「登录发生在拿到令牌之前」的硬约束。
// 之前三条内联在 staff.go 里靠人工保证，现纳入声明后由本测试与 endpoint.Mount 的语义共同把守。
func TestStaffPublicExtraNotGuarded(t *testing.T) {
	pub := staffPublicExtra(nil)
	if len(pub) != 3 {
		t.Fatalf("员工端登录出口 %d 条，期望 3", len(pub))
	}
	seen := map[string]bool{}
	for _, r := range pub {
		if r.Perm != "" {
			t.Errorf("登录出口 %s 挂了权限点 %q：登录时尚未认证，权限点无法判定", r.Path, r.Perm)
		}
		if r.Handler == nil {
			t.Errorf("登录出口 %s 的 Handler 为空", r.Path)
		}
		if seen[r.Path] {
			t.Errorf("登录出口路径重复: %s", r.Path)
		}
		seen[r.Path] = true
	}

	// 免中间件分组下必须能注册（不得因中间件缺失而 panic）
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	ep := staffEndpoint(nil, nil, nil)
	endpoint.Mount(engine.Group(""), ep, routes.Common(nil), nil)
	n := 0
	for _, ri := range engine.Routes() {
		if strings.HasPrefix(ri.Path, "/wechat/staff/auth/") {
			n++
		}
	}
	if n != 3 {
		t.Errorf("免中间件分组下注册了 %d 条登录出口，期望 3", n)
	}
}
