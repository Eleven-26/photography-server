package dto

import (
	"testing"

	"photography-server/internal/model"
)

// TestNewStaffStudioSettingResp_HomepageURL 覆盖 homepage_url 的拼装边界。
// 该字段是员工端「我的预约主页」展示与复制的分享链接（客户在微信内打开落到 H5 预约主页），
// 拼错会让客户打开到错误域名、或把订单归错员工，故对空值、尾斜杠、转义、staff_id 缺省
// 四类易错点做固定断言。
func TestNewStaffStudioSettingResp_HomepageURL(t *testing.T) {
	cases := []struct {
		name    string
		slug    string
		base    string
		staffID int64
		want    string
	}{
		{"正常拼装（含分享人）", "lusheng-photography", "https://slot.app", 12, "https://slot.app/?slug=lusheng-photography&staff_id=12"},
		{"分享人缺省（0）→ 省略 staff_id", "lusheng-photography", "https://slot.app", 0, "https://slot.app/?slug=lusheng-photography"},
		{"分享人为负值 → 视同缺省", "a", "https://slot.app", -1, "https://slot.app/?slug=a"},
		{"基址带尾斜杠归一化", "lusheng-photography", "https://slot.app/", 12, "https://slot.app/?slug=lusheng-photography&staff_id=12"},
		{"基址带多个尾斜杠", "a", "https://slot.app///", 0, "https://slot.app/?slug=a"},
		{"基址两侧空白与尾斜杠同时存在", "a", "  https://slot.app/  ", 0, "https://slot.app/?slug=a"},
		{"slug 为空 → 不产出链接（即便有分享人）", "", "https://slot.app", 12, ""},
		{"基址为空 → 不产出链接", "lusheng-photography", "", 12, ""},
		{"两侧均空 → 空串", "", "", 0, ""},
		{"slug 带两侧空白被裁剪", "  lusheng-photography  ", "https://slot.app", 0, "https://slot.app/?slug=lusheng-photography"},
		{"slug 含需转义字符（空格与斜杠）", "a b/c", "https://slot.app", 0, "https://slot.app/?slug=a+b%2Fc"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := NewStaffStudioSettingResp(&model.StudioSetting{HomepageSlug: c.slug}, c.base, c.staffID)
			if got.HomepageURL != c.want {
				t.Errorf("homepage_url = %q, want %q", got.HomepageURL, c.want)
			}
		})
	}
}

// TestNewStaffStudioSettingResp_NilSetting 设置行为 nil 时不得 panic，且不产出链接。
func TestNewStaffStudioSettingResp_NilSetting(t *testing.T) {
	got := NewStaffStudioSettingResp(nil, "https://slot.app", 12)
	if got == nil {
		t.Fatal("st 为 nil 时不应返回 nil 响应")
	}
	if got.HomepageURL != "" {
		t.Errorf("st 为 nil 时 homepage_url 应为空串，得到 %q", got.HomepageURL)
	}
}

// TestNewStaffStudioSettingResp_EmbedsSetting 确认嵌入的 model.StudioSetting 被保留，
// 序列化时其字段会与 homepage_url 平铺在同一层（员工端一次请求拿到设置项 + 分享链接）。
func TestNewStaffStudioSettingResp_EmbedsSetting(t *testing.T) {
	st := &model.StudioSetting{HomepageSlug: "slot-demo", Slogan: "用光影记录值得珍藏的瞬间"}
	got := NewStaffStudioSettingResp(st, "https://slot.app", 7)
	if got.StudioSetting != st {
		t.Error("嵌入的 StudioSetting 指针应原样保留")
	}
	if got.Slogan != "用光影记录值得珍藏的瞬间" {
		t.Errorf("嵌入字段未提升，Slogan = %q", got.Slogan)
	}
	if got.HomepageURL != "https://slot.app/?slug=slot-demo&staff_id=7" {
		t.Errorf("嵌入场景下 homepage_url 仍应带分享人，得到 %q", got.HomepageURL)
	}
}
