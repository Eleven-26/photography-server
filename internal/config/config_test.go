package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeConf 写临时配置文件（bootstrap），返回完整路径
func writeConf(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("写配置文件失败: %v", err)
	}
	return p
}

// Nacos 单一配置源的加载语义测试：
// 本地文件仅为 bootstrap（nacos 连接段），业务配置 100% 来自远程（fake fetcher 模拟），
// 优先级 APP_* env > 远程 Nacos > 本地 bootstrap；拉取失败 fail-fast。

// bootstrapConf 模拟瘦身后 config/config.yaml：仅 nacos 连接段 + 一个业务 key 用于对比覆盖
const bootstrapConf = `
app:
  name: demo
  port: 8080
jwt:
  issuer: local-issuer
nacos:
  server_addr: 127.0.0.1:8848
  data_id: photography-server-${profile}.yaml
  register_ip: ""
  timeout_ms: 5000
`

// remoteConf 模拟 Nacos 控制台发布的业务配置
const remoteConf = `
jwt:
  secret: remote-secret
  issuer: remote-issuer
db:
  host: remote-db
  user: root
  password: root
log:
  level: warn
`

func fakeFetcher(content string) Fetcher {
	return func(n Nacos) (string, error) { return content, nil }
}

func writeBootstrap(t *testing.T, content string) string {
	t.Helper()
	return writeConf(t, t.TempDir(), "config.yaml", content)
}

// 远程配置覆盖 bootstrap 同名 key，未覆盖 key 保留 bootstrap 值
func TestLoadRemoteOverridesBootstrap(t *testing.T) {
	base := writeBootstrap(t, bootstrapConf)
	cfg, err := LoadWithFetcher(base, "dev", fakeFetcher(remoteConf))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.JWT.Secret != "remote-secret" {
		t.Fatalf("远程应覆盖 bootstrap: jwt.secret=%q", cfg.JWT.Secret)
	}
	if cfg.JWT.Issuer != "remote-issuer" {
		t.Fatalf("远程值未生效: jwt.issuer=%q", cfg.JWT.Issuer)
	}
	if cfg.DB.Host != "remote-db" {
		t.Fatalf("远程业务配置未加载: db.host=%q", cfg.DB.Host)
	}
}

// 优先级：APP_* env > 远程 Nacos > 本地 bootstrap
func TestLoadEnvAboveRemote(t *testing.T) {
	base := writeBootstrap(t, bootstrapConf)
	t.Setenv("APP_LOG_LEVEL", "debug")
	cfg, err := LoadWithFetcher(base, "dev", fakeFetcher(remoteConf))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Log.Level != "debug" {
		t.Fatalf("env 应优先于远程: log.level=%q want debug", cfg.Log.Level)
	}
}

// fetch 返回错误时启动终止（fail-fast），不允许带着缺失配置上线
func TestLoadFetchErrorFailFast(t *testing.T) {
	base := writeBootstrap(t, bootstrapConf)
	fetch := func(n Nacos) (string, error) { return "", fmt.Errorf("nacos 不可达") }
	if _, err := LoadWithFetcher(base, "dev", fetch); err == nil {
		t.Fatal("拉取失败应返回错误终止启动，实际通过")
	}
}

// 远程配置为空时视为发布缺失，fail-fast（与 SDK 行为一致的防线）
func TestLoadRemoteEmptyFailFast(t *testing.T) {
	base := writeBootstrap(t, bootstrapConf)
	if _, err := LoadWithFetcher(base, "dev", fakeFetcher("")); err == nil {
		t.Fatal("远程配置为空应返回错误，实际通过")
	}
}

// data_id 的 ${profile} 占位按 -p/APP_PROFILE 替换
func TestLoadDataIdProfilePlaceholder(t *testing.T) {
	base := writeBootstrap(t, bootstrapConf)
	var got Nacos
	fetch := func(n Nacos) (string, error) {
		got = n
		return remoteConf, nil
	}
	if _, err := LoadWithFetcher(base, "prod", fetch); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if want := "photography-server-prod.yaml"; got.DataId != want {
		t.Fatalf("data_id 占位未替换: %q want %q", got.DataId, want)
	}
}

// fetch 为空时跳过远程（仅供测试/工具），bootstrap 内业务 key 直接生效
func TestLoadWithoutFetcher(t *testing.T) {
	base := writeBootstrap(t, bootstrapConf)
	cfg, err := Load(base, "dev")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.JWT.Issuer != "local-issuer" {
		t.Fatalf("bootstrap 值未生效: jwt.issuer=%q", cfg.JWT.Issuer)
	}
	if cfg.JWT.Secret != "" {
		t.Fatalf("无 fetcher 时不应有业务配置: jwt.secret=%q", cfg.JWT.Secret)
	}
}

// prod 校验：远程配置缺失必注入项时快速失败
func TestLoadProdValidation(t *testing.T) {
	base := writeBootstrap(t, bootstrapConf)
	remote := strings.Replace(remoteConf, "remote-secret", "", 1)
	if _, err := LoadWithFetcher(base, "prod", fakeFetcher(remote)); err == nil {
		t.Fatal("prod 空 secret 应当校验失败，实际通过")
	}
}

// prod 校验：默认弱密钥被拒绝；dev 不做校验
func TestLoadProdRejectsDefaultSecret(t *testing.T) {
	base := writeBootstrap(t, bootstrapConf)
	remote := strings.Replace(remoteConf, "remote-secret", defaultJWTSecret, 1)
	if _, err := LoadWithFetcher(base, "prod", fakeFetcher(remote)); err == nil {
		t.Fatal("prod 默认弱密钥应当校验失败，实际通过")
	}
	if _, err := LoadWithFetcher(base, "dev", fakeFetcher(remote)); err != nil {
		t.Fatalf("dev 不应触发校验: %v", err)
	}
}
