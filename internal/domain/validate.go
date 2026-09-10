package domain

import "regexp"

// mobileRe 中国大陆手机号：1 开头，第二位 3-9，共 11 位。
// 客户登录以「手机号 + 验证码」为凭据，换绑手机号前必须过这一关。
var mobileRe = regexp.MustCompile(`^1[3-9]\d{9}$`)

// IsMobile 校验手机号格式（中国大陆）
func IsMobile(s string) bool {
	return mobileRe.MatchString(s)
}
