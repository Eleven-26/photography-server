package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"photography-server/internal/pkg/logger"

	"github.com/spf13/viper"
)

type Config struct {
	App    App    `mapstructure:"app"`
	JWT    JWT    `mapstructure:"jwt"`
	DB     DB     `mapstructure:"db"`
	Redis  Redis  `mapstructure:"redis"`
	NATS   NATS   `mapstructure:"nats"`
	Mongo  Mongo  `mapstructure:"mongodb"`
	Log    Log    `mapstructure:"log"`
	Upload Upload `mapstructure:"upload"`
	XxlJob XxlJob `mapstructure:"xxljob"`
	ES     ES     `mapstructure:"elasticsearch"`
	Jaeger Jaeger `mapstructure:"jaeger"`
	Nacos  Nacos  `mapstructure:"nacos"`
}

type ES struct {
	Enable   bool     `mapstructure:"enable"`
	Urls     []string `mapstructure:"urls"`
	Username string   `mapstructure:"username"`
	Password string   `mapstructure:"password"`
}

// Jaeger Jaeger 链路通道配置（复用 OpenTelemetry 埋点，OTLP gRPC 上报 Jaeger，存储 ClickHouse）。
// 本段实为通用 OTel exporter 开关/地址，当前唯一用途即 Jaeger 通道：
//
//	Enable=true 时 OTel SDK 经 OTLP gRPC 上报 Endpoint（如 jaeger:4317）。
//	两通道各自独立、互不依赖，但勿同时开启（同一请求会产双 span/双上报）：
//	  通道① SkyWalking-go(native)：Dockerfile SW_AGENT_ENABLE=true 构建期注入 agent，
//	      直连 OAP:11800（agent.config 的 backend_service），无运行时开关，与本段无关；
//	  通道② OTel→Jaeger：本段 enable=true + endpoint=jaeger:4317，不注入 agent，
//	      复用全部 OTel 手动埋点（HTTP 入口/SQL/xxl-job/NATS），Jaeger UI 按 trace_id 查。
type Jaeger struct {
	Enable   bool   `mapstructure:"enable"`
	Endpoint string `mapstructure:"endpoint"` // OTLP gRPC endpoint：jaeger:4317（compose 服务名）
	Service  string `mapstructure:"service"`  // 上报的服务名（otelgin/span 的 service.name）
	Instance string `mapstructure:"instance"` // 实例名，留空默认取主机名
}

// Nacos 配置中心 + 服务注册发现。一个开关（enable）切换两种模式：
//
//	enable=true（部署环境）：本地 config.yaml 退化为 bootstrap —— 只提供本段 nacos.* 与兜底默认值，
//	  业务配置以 Nacos 上 data_id 对应的 YAML 为准（远程合并到本地之上，远程优先）。
//	  拉取失败直接返回错误终止启动（fail-fast，避免带着错误配置上线）；
//	  Nacos 整体不可用时由 SDK 降级读取本地快照（cache_dir）尝试兜底。
//	  配置变更需重启进程生效（不做运行期热更）。
//	  同时在启动时把本机 IP:Port 注册为临时实例（SDK 自动心跳），优雅退出时反注册。
//	enable=false（本地开发）：完全沿用本地三层加载，行为与引入 Nacos 前一致，无需起 Nacos。
type Nacos struct {
	Enable     bool   `mapstructure:"enable"`
	ServerAddr string `mapstructure:"server_addr"` // host:8848；v2 SDK 走 gRPC，端口自动 +1000（9848）
	Namespace  string `mapstructure:"namespace"`   // 命名空间 ID，留空=public
	Group      string `mapstructure:"group"`       // 默认 DEFAULT_GROUP
	DataId     string `mapstructure:"data_id"`     // 配置 dataId，支持 ${profile} 占位符
	Username   string `mapstructure:"username"`    // 开启鉴权时必填，否则留空
	Password   string `mapstructure:"password"`
	TimeoutMs  uint64 `mapstructure:"timeout_ms"` // 请求超时，默认 5000
	CacheDir   string `mapstructure:"cache_dir"`  // 本地快照目录，默认 .nacos/cache
	LogLevel   string `mapstructure:"log_level"`  // SDK 自身日志级别，默认 warn
	// ---- 服务注册 ----
	ServiceName string  `mapstructure:"service_name"` // 注册的服务名，留空取 app.name
	ClusterName string  `mapstructure:"cluster_name"` // 集群名，默认 DEFAULT
	Weight      float64 `mapstructure:"weight"`       // 权重，默认 1
	RegisterIp  string  `mapstructure:"register_ip"`  // 注册 IP，留空自动探测（容器内为容器 IP）
}

// withDefaults 填默认值并把 data_id 里的 ${profile} 占位替换为实际环境名
func (n Nacos) withDefaults(profile string) Nacos {
	if n.Group == "" {
		n.Group = "DEFAULT_GROUP"
	}
	if n.DataId == "" {
		n.DataId = "photography-server.yaml"
	}
	n.DataId = strings.ReplaceAll(n.DataId, "${profile}", profile)
	if n.TimeoutMs == 0 {
		n.TimeoutMs = 5000
	}
	if n.CacheDir == "" {
		n.CacheDir = filepath.Join(".nacos", "cache")
	}
	if n.LogLevel == "" {
		n.LogLevel = "warn"
	}
	if n.ClusterName == "" {
		n.ClusterName = "DEFAULT"
	}
	if n.Weight <= 0 {
		n.Weight = 1
	}
	return n
}

// Normalized 返回填好默认值的副本（供基础设施层使用，避免各处重复兜底逻辑）
func (n Nacos) Normalized(profile string) Nacos { return n.withDefaults(profile) }

type XxlJob struct {
	Enable       bool   `mapstructure:"enable"`
	ServerAddr   string `mapstructure:"server_addr"`
	AccessToken  string `mapstructure:"access_token"`
	ExecutorIp   string `mapstructure:"executor_ip"`
	ExecutorPort string `mapstructure:"executor_port"`
	RegistryKey  string `mapstructure:"registry_key"`
	LogDir       string `mapstructure:"log_dir"`
}

type App struct {
	Name     string `mapstructure:"name"`
	Mode     string `mapstructure:"mode"`
	Port     int    `mapstructure:"port"`
	Timezone string `mapstructure:"timezone"`
	Profile  string `mapstructure:"profile"`
}

type JWT struct {
	Secret      string `mapstructure:"secret"`
	Issuer      string `mapstructure:"issuer"`
	ExpireHours int    `mapstructure:"expire_hours"`
}

type DB struct {
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	User         string `mapstructure:"user"`
	Password     string `mapstructure:"password"`
	Name         string `mapstructure:"name"`
	Charset      string `mapstructure:"charset"`
	MaxIdleConns int    `mapstructure:"max_idle_conns"`
	MaxOpenConns int    `mapstructure:"max_open_conns"`
	LogMode      bool   `mapstructure:"log_mode"`
}

func (d DB) DSN() string {
	return strings.Join([]string{
		d.User + ":" + d.Password,
		"@tcp(" + d.Host + ":" + strconv.Itoa(d.Port) + ")/" + d.Name,
		"?charset=" + d.Charset + "&parseTime=True&loc=Local",
	}, "")
}

type Redis struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type NATS struct {
	URL string `mapstructure:"url"`
}

type Mongo struct {
	Enable   bool   `mapstructure:"enable"`
	URI      string `mapstructure:"uri"`
	Database string `mapstructure:"database"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
}

type Log struct {
	Level string `mapstructure:"level"`
}

type Upload struct {
	Dir       string `mapstructure:"dir"`
	MaxSizeMB int    `mapstructure:"max_size_mb"`
}

// Fetcher 从远程配置中心拉取配置内容（YAML 文本），入参为本地 bootstrap 解析出的 Nacos 段。
// 由 main 注入基础设施层实现（infrastructure.FetchConfig），避免 config 反向依赖 infrastructure。
type Fetcher func(n Nacos) (string, error)

// Load 纯本地加载，等价于 LoadWithFetcher(base, profile, nil)
func Load(basePath, profile string) (*Config, error) {
	return LoadWithFetcher(basePath, profile, nil)
}

// LoadWithFetcher 加载配置：
// 基础配置 config.yaml → 环境覆盖 config.<profile>.yaml →（nacos.enable 时）远程配置 → APP_* 环境变量
// 优先级：APP_* 环境变量 > 远程配置（Nacos）> config.<profile>.yaml > config.yaml
// 注意：不做 ${VAR} 模板展开；Unmarshal 只能覆盖配置文件中已存在的 key，新增配置项需同步维护各 yaml。
func LoadWithFetcher(basePath, profile string, fetch Fetcher) (*Config, error) {
	if profile == "" {
		profile = "dev"
	}
	profile = strings.ToLower(profile)

	v := viper.New()
	v.SetConfigFile(basePath)
	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}

	// 环境专用配置覆盖基础配置
	ppath := profilePath(basePath, profile)
	if _, err := os.Stat(ppath); err == nil {
		v.SetConfigFile(ppath)
		if err := v.MergeInConfig(); err != nil {
			return nil, err
		}
	}

	// 环境变量（APP_ 前缀）优先于配置文件
	// AutomaticEnv 会在 Get 时自动查找 APP_ 前缀的环境变量
	// 例如 v.GetString("db.host") → 查找 APP_DB_HOST
	v.SetEnvPrefix("APP")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// 第一阶段解析：只为拿到 nacos 段，判断是否需要从配置中心拉取
	var boot Config
	if err := v.Unmarshal(&boot); err != nil {
		return nil, err
	}
	if fetch != nil && boot.Nacos.Enable {
		n := boot.Nacos.withDefaults(profile)
		content, err := fetch(n)
		if err != nil {
			return nil, fmt.Errorf("拉取 Nacos 配置失败（data_id=%s group=%s）: %w", n.DataId, n.Group, err)
		}
		v.SetConfigType("yaml")
		if err := v.MergeConfig(strings.NewReader(content)); err != nil {
			return nil, fmt.Errorf("解析 Nacos 配置失败（data_id=%s）: %w", n.DataId, err)
		}
		logger.Infof("已加载配置中心远端配置: data_id=%s group=%s", n.DataId, n.Group)
	}

	// profile 由启动参数/环境变量决定，远程配置不得覆盖
	v.Set("app.profile", profile)

	var c Config
	if err := v.Unmarshal(&c); err != nil {
		return nil, err
	}
	if c.App.Port == 0 {
		c.App.Port = 8080
	}
	if c.JWT.ExpireHours == 0 {
		c.JWT.ExpireHours = 168
	}
	if c.Upload.MaxSizeMB == 0 {
		c.Upload.MaxSizeMB = 20
	}
	if err := validateProd(&c); err != nil {
		return nil, err
	}
	return &c, nil
}

// defaultJWTSecret 与 config.yaml 中的占位密钥一致，prod 校验拒绝该弱默认值
const defaultJWTSecret = "photography-server-jwt-secret-change-me"

// validateProd prod 安全基线：必注入项缺失时快速失败，避免空凭据连库或弱密钥上线
func validateProd(c *Config) error {
	if c.App.Profile != "prod" {
		return nil
	}
	if c.JWT.Secret == "" || c.JWT.Secret == defaultJWTSecret {
		return errors.New("prod 环境必须通过 APP_JWT_SECRET 注入 JWT 密钥（不得为空或默认值）")
	}
	if c.DB.Host == "" || c.DB.User == "" || c.DB.Password == "" {
		return errors.New("prod 环境必须通过 APP_DB_HOST / APP_DB_USER / APP_DB_PASSWORD 注入数据库连接信息")
	}
	return nil
}

// profilePath 计算环境配置文件名：config/config.yaml + prod => config/config.prod.yaml
func profilePath(base, profile string) string {
	dir := filepath.Dir(base)
	name := filepath.Base(base)
	ext := filepath.Ext(name)
	stem := strings.TrimSuffix(name, ext)
	return filepath.Join(dir, stem+"."+profile+ext)
}
