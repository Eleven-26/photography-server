package router

import (
	"testing"

	"github.com/gin-gonic/gin"
)

// TestRegisterSwaggerGating Swagger UI 注册开关的护栏：
// 仅在 dev/test/docker.dev 注册 /swagger/*any，prod 及未知 profile 一律不注册。
// 防止有人把文档入口误开到生产（内部接口清单对外暴露）。
func TestRegisterSwaggerGating(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cases := []struct {
		profile string
		want    bool
	}{
		{"dev", true},
		{"test", true},
		{"docker.dev", true},
		{"prod", false},
		{"", false},
		{"staging", false},
	}
	for _, tc := range cases {
		engine := gin.New()
		registerSwagger(engine, tc.profile)

		found := false
		for _, ri := range engine.Routes() {
			if ri.Method == "GET" && ri.Path == "/swagger/*any" {
				found = true
				break
			}
		}
		if found != tc.want {
			t.Errorf("profile=%q 注册 /swagger/*any = %v, want %v", tc.profile, found, tc.want)
		}
	}
}
