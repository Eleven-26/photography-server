package router

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"

	"photography-server/internal/service"
)

// TestPublicMediaStaticRoute 静态目录注册的护栏（2026-09-14 新增 /media 时补）。
//
// 两条 catch-all 必须共存且互不遮蔽：
//   - /uploads/*filepath → 挂 AssetAuth，客户隐私文件（样片/成片/凭证）
//   - /media/*filepath   → **不挂鉴权**，公开作品图（作品集封面/图集）
//
// gin 在路由树构建期遇到冲突会直接 panic（服务根本起不来），所以本测试首先保证
// "注册不炸"；其次锁定 /media 确实匿名可读 —— 否则 H5 分享页的未登录浏览者会看到一片白。
// 目录名与 URL 根路径由 service.PublicMediaDir 同源推出（router.go 亦引用该常量）。
func TestPublicMediaStaticRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)

	root := t.TempDir()
	mediaRoot := filepath.Join(root, service.PublicMediaDir)
	if err := os.MkdirAll(filepath.Join(mediaRoot, "202609"), 0o755); err != nil {
		t.Fatalf("准备公开目录失败: %v", err)
	}
	// 同一份文件也放在鉴权目录下，用于对照两条路由的鉴权差异
	if err := os.MkdirAll(filepath.Join(root, "202609"), 0o755); err != nil {
		t.Fatalf("准备鉴权目录失败: %v", err)
	}
	for _, p := range []string{
		filepath.Join(mediaRoot, "202609", "a.jpg"),
		filepath.Join(root, "202609", "a.jpg"),
	} {
		if err := os.WriteFile(p, []byte("img"), 0o644); err != nil {
			t.Fatalf("写入测试文件失败: %v", err)
		}
	}

	engine := gin.New()
	// 与 router.go 保持同一注册方式：鉴权组的 catch-all + 公开目录的静态路由
	engine.Group("/uploads", func(c *gin.Context) { c.AbortWithStatus(http.StatusUnauthorized) }).
		Static("/", root)
	engine.Static("/"+service.PublicMediaDir, mediaRoot)

	t.Run("公开媒体匿名可读", func(t *testing.T) {
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/media/202609/a.jpg", nil))
		if w.Code != http.StatusOK {
			t.Errorf("状态 = %d, want 200；未登录浏览者会看不到作品图", w.Code)
		}
		if body := w.Body.String(); body != "img" {
			t.Errorf("响应体 = %q, want %q", body, "img")
		}
	})

	t.Run("鉴权目录仍被拦截", func(t *testing.T) {
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/uploads/202609/a.jpg", nil))
		if w.Code != http.StatusUnauthorized {
			t.Errorf("状态 = %d, want 401；放开 /media 不得顺带放开 /uploads", w.Code)
		}
	})

	t.Run("鉴权目录下的公开子目录也不暴露", func(t *testing.T) {
		// /uploads/media/... 走的是鉴权 catch-all，必须仍被拦截
		if err := os.MkdirAll(filepath.Join(root, service.PublicMediaDir), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, service.PublicMediaDir, "b.jpg"), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/uploads/media/b.jpg", nil))
		if w.Code != http.StatusUnauthorized {
			t.Errorf("状态 = %d, want 401", w.Code)
		}
	})
}
