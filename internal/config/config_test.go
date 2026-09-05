package config

import (
	"os"
	"path/filepath"
	"testing"
)

const baseConf = `
app:
  name: demo
  port: 8080
jwt:
  secret: test-secret
  issuer: demo
db:
  host: 127.0.0.1
  user: root
  password: root
log:
  level: info
`

func writeConf(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("写配置文件失败: %v", err)
	}
	return p
}

// 环境文件覆盖基础配置，未覆盖项回落基础值
func TestLoadProfileOverride(t *testing.T) {
	dir := t.TempDir()
	base := writeConf(t, dir, "config.yaml", baseConf)
	writeConf(t, dir, "config.prod.yaml", "log:\n  level: warn\nupload:\n  max_size_mb: 100\n")

	cfg, err := Load(base, "prod")
	if err != nil {
		t.Fatalf("Load 失败: %v", err)
	}
	if cfg.Log.Level != "warn" {
		t.Fatalf("环境覆盖未生效: log.level = %q, want warn", cfg.Log.Level)
	}
	if cfg.JWT.Issuer != "demo" {
		t.Fatalf("基础配置丢失: jwt.issuer = %q, want demo", cfg.JWT.Issuer)
	}
	if cfg.Upload.MaxSizeMB != 100 {
		t.Fatalf("环境覆盖未生效: upload.max_size_mb = %d, want 100", cfg.Upload.MaxSizeMB)
	}
	if cfg.App.Profile != "prod" {
		t.Fatalf("profile 未写入: %q", cfg.App.Profile)
	}
}

// APP_* 环境变量优先于配置文件
func TestLoadEnvOverride(t *testing.T) {
	dir := t.TempDir()
	base := writeConf(t, dir, "config.yaml", baseConf)
	writeConf(t, dir, "config.prod.yaml", "log:\n  level: warn\n")
	t.Setenv("APP_LOG_LEVEL", "debug")

	cfg, err := Load(base, "prod")
	if err != nil {
		t.Fatalf("Load 失败: %v", err)
	}
	if cfg.Log.Level != "debug" {
		t.Fatalf("环境变量覆盖未生效: log.level = %q, want debug", cfg.Log.Level)
	}
}

// prod 缺失必注入项时快速失败
func TestLoadProdValidation(t *testing.T) {
	dir := t.TempDir()
	base := writeConf(t, dir, "config.yaml", baseConf)
	writeConf(t, dir, "config.prod.yaml", "jwt:\n  secret: \"\"\n")

	if _, err := Load(base, "prod"); err == nil {
		t.Fatal("prod 空 secret 应当校验失败，实际通过")
	}
}

// 弱默认密钥在 prod 被拒绝；dev 不做校验
func TestLoadProdRejectsDefaultSecret(t *testing.T) {
	dir := t.TempDir()
	base := writeConf(t, dir, "config.yaml", `
app:
  name: demo
jwt:
  secret: photography-server-jwt-secret-change-me
db:
  host: ""
  user: root
  password: root
`)
	if _, err := Load(base, "prod"); err == nil {
		t.Fatal("prod 默认弱密钥应当校验失败，实际通过")
	}
	if _, err := Load(base, "dev"); err != nil {
		t.Fatalf("dev 不应触发校验: %v", err)
	}
}
