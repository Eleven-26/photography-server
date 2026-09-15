package h5

import (
	"testing"

	"photography-server/internal/config"
)

// 免验证码登录是**安全敏感开关**：一旦在生产被打开，等于「知道手机号即可登录该客户账号」，
// 可读其订单、交付样片、评价与个人资料。本测试把「只有开发环境放开」这条约束钉死，
// 防止日后有人顺手把它改成 `return false`、或用配置项替换白名单判断。
func TestLoginRequireSmsCode(t *testing.T) {
	cases := []struct {
		profile string
		want    bool
	}{
		{"dev", false},
		{"docker.dev", false},
		{"test", true},    // 测试环境不放开：短信通道虽未接入，但只认显式开发环境
		{"prod", true},    // 生产必须验证码
		{"", true},        // profile 缺失按最严处理（config 层会兜底成 dev，此处只锁「默认拒绝」语义）
		{"staging", true}, // 未知 profile 一律强制，避免新环境被默默放开
	}
	for _, c := range cases {
		ctl := &Controller{Cfg: &config.Config{App: config.App{Profile: c.profile}}}
		if got := ctl.loginRequireSmsCode(); got != c.want {
			t.Errorf("profile=%q: loginRequireSmsCode()=%v, want %v", c.profile, got, c.want)
		}
	}
}
