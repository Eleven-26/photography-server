package router

import (
	"reflect"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"photography-server/internal/presentation/endpoint"
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
}

// staffRouteCountBaseline 员工端路由数基线：
// 26（与 PC 同路径复用，含 2026-09-12 补开的 4 条 /notification/*）
// + 6（异路径别名）+ 2（真实端差异）+ 18（移动端独有）= 52。
// 有意增减员工端暴露面时须同步更新此基线——它防的是"悄悄多挂 / 漏挂"。
const staffRouteCountBaseline = 52

// commonRouteCountBaseline 管理端业务路由数基线（PC 与小程序共用）。
const commonRouteCountBaseline = 108

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
	for _, r := range extra {
		if seen[r.Path] {
			t.Errorf("员工端路由重复注册（会导致 gin 冲突）: %s", r.Path)
		}
		seen[r.Path] = true
	}

	// 3) 路由数基线
	if got := len(staffInclude) + len(extra); got != staffRouteCountBaseline {
		t.Errorf("员工端路由数 %d，期望 %d", got, staffRouteCountBaseline)
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
