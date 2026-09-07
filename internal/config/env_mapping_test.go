package config

import (
	"testing"
)

// env_mapping_test.go 固化 viper AutomaticEnv 的变量名映射约定（曾因命名错误产生过一批死变量）：
// 变量名 = APP_ + 段名_键名（EnvKeyReplacer 仅把 "." 替换为 "_"，段名参与拼接）。
// 断言「正确名生效」与「旧错误名不生效」双向，防止回归。

const envMapBase = `
app:
  name: demo
  mode: debug
  port: 8080
redis:
  addr: 127.0.0.1:6379
mongodb:
  enable: false
  uri: mongodb://127.0.0.1:27017
  database: photography
elasticsearch:
  enable: false
  urls:
    - http://127.0.0.1:9200
jaeger:
  enable: false
  endpoint: 127.0.0.1:4317
nacos:
  enable: false
  register_ip: ""
  timeout_ms: 5000
`

func TestEnvVarNameMapping(t *testing.T) {
	dir := t.TempDir()
	base := writeConf(t, dir, "config.yaml", envMapBase)

	// 正确名（生效）+ 旧错误名（不得生效）
	t.Setenv("APP_MONGODB_ENABLE", "true")
	t.Setenv("APP_MONGODB_URI", "mongodb://probed:27017")
	t.Setenv("APP_MONGO_URI", "mongodb://dead:27017") // 旧名，必须无效
	t.Setenv("APP_ELASTICSEARCH_ENABLE", "true")
	t.Setenv("APP_ELASTICSEARCH_URLS", "http://a:9200,http://b:9200") // 逗号切分为切片
	t.Setenv("APP_ES_ENABLE", "true")                                 // 旧名，必须无效
	t.Setenv("APP_REDIS_ADDR", "probed:6379")
	t.Setenv("APP_REDIS_HOST", "dead") // 旧名，必须无效
	t.Setenv("APP_APP_MODE", "release")
	t.Setenv("APP_MODE", "debug") // 旧名，必须无效
	t.Setenv("APP_NACOS_REGISTER_IP", "10.0.0.9")
	t.Setenv("APP_NACOS_TIMEOUT_MS", "8000")

	cfg, err := Load(base, "dev")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if !cfg.Mongo.Enable || cfg.Mongo.URI != "mongodb://probed:27017" {
		t.Errorf("mongodb 段映射错误: enable=%v uri=%q", cfg.Mongo.Enable, cfg.Mongo.URI)
	}
	if !cfg.ES.Enable {
		t.Errorf("APP_ELASTICSEARCH_ENABLE 未生效")
	}
	if len(cfg.ES.Urls) != 2 || cfg.ES.Urls[0] != "http://a:9200" {
		t.Errorf("APP_ELASTICSEARCH_URLS 逗号切分未生效: %v", cfg.ES.Urls)
	}
	if cfg.Redis.Addr != "probed:6379" {
		t.Errorf("APP_REDIS_ADDR 未生效: %q", cfg.Redis.Addr)
	}
	if cfg.App.Mode != "release" {
		t.Errorf("APP_APP_MODE 未生效: %q", cfg.App.Mode)
	}
	if cfg.Nacos.RegisterIp != "10.0.0.9" || cfg.Nacos.TimeoutMs != 8000 {
		t.Errorf("nacos 可选字段 env 覆盖未生效: ip=%q timeout=%d", cfg.Nacos.RegisterIp, cfg.Nacos.TimeoutMs)
	}
}
