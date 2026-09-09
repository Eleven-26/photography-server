package config

import (
	"errors"
	"fmt"
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

// Nacos 配置中心 + 服务注册发现（唯一业务配置源，硬依赖，无开关）：
//
//	本地 config.yaml 仅为 bootstrap —— 只提供本段 nacos.* 连接信息，
//	业务配置 100% 以 Nacos 上 data_id 对应的 YAML 为准（data_id 支持 ${profile} 按环境区分，
//	发布内容模板见 config/nacos/）。
//	拉取失败直接返回错误终止启动（fail-fast，避免带着错误配置上线）；
//	Nacos 整体不可用时由 SDK 降级读取本地快照（cache_dir）尝试兜底。
//	配置变更需重启进程生效（不做运行期热更）。
//	启动成功后把本机 IP:Port 注册为临时实例（SDK 自动心跳），优雅退出时反注册。
type Nacos struct {
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
	Name        string   `mapstructure:"name"`
	Mode        string   `mapstructure:"mode"`
	Port        int      `mapstructure:"port"`
	Timezone    string   `mapstructure:"timezone"`
	Profile     string   `mapstructure:"profile"`
	CORSOrigins []string `mapstructure:"cors_origins"` // CORS 可信来源白名单（精确 Origin，如 http://localhost:5173）
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

// LoadWithFetcher 加载配置（Nacos 唯一业务配置源）：
// 本地 bootstrap config.yaml（仅 nacos 连接段）→ 必拉远程配置（data_id 按 profile 区分）→ APP_* 环境变量
// 优先级：APP_* 环境变量 > 远程配置（Nacos）> 本地 bootstrap
// fetch 为空时跳过远程拉取（仅供测试/工具使用，业务字段将缺失）；生产路径必须注入 fetcher（main 传 infrastructure.FetchConfig）。
// 注意：不做 ${VAR} 模板展开；远程未发布的 key 即为零值（仅 app.port / jwt.expire_hours / upload.max_size_mb 有代码兜底）。
func LoadWithFetcher(basePath, profile string, fetch Fetcher) (*Config, error) {
	if profile == "" {
		profile = "dev"
	}
	profile = strings.ToLower(profile)

	// 主密钥（KEK）：来自 APP_CONFIG_SECRET_FILE / APP_CONFIG_SECRET，不进 Nacos、不进 git。
	// 未配置时不启用解密（dev 明文模板可正常启动）；配置中没有 ENCv1 密文也不会用到它。
	cipher, err := LoadCipher()
	if err != nil {
		return nil, err
	}

	v := viper.New()
	v.SetConfigFile(basePath)
	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}

	// 环境变量（APP_ 前缀）优先于配置文件
	// AutomaticEnv 会在 Get 时自动查找 APP_ 前缀的环境变量
	// 例如 v.GetString("db.host") → 查找 APP_DB_HOST
	v.SetEnvPrefix("APP")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// 第一阶段解析：只为拿到 bootstrap 里的 nacos 段（连接信息，可被 APP_NACOS_* env 覆盖）
	var boot Config
	if err := v.Unmarshal(&boot); err != nil {
		return nil, err
	}
	// 远程配置是唯一业务配置源：fetch 非空必拉，失败即终止启动（fail-fast）；
	// Nacos 整体不可达时由 SDK 自动降级读本地快照（cache_dir），快照也没有才返回错误
	if fetch != nil {
		n := boot.Nacos.withDefaults(profile)
		content, err := fetch(n)
		if err != nil {
			msg := fmt.Sprintf("拉取 Nacos 配置失败（data_id=%s group=%s）: %v", n.DataId, n.Group, err)
			if n.Username == "" {
				msg += "；当前 nacos.username 为空——服务端开启鉴权时请用 APP_NACOS_USERNAME/APP_NACOS_PASSWORD 注入（本地运行用 make run-dev 自动加载 .env）"
			}
			return nil, errors.New(msg)
		}
		if strings.TrimSpace(content) == "" {
			return nil, fmt.Errorf("远端配置为空（data_id=%s group=%s），请检查 Nacos 控制台是否已发布该配置", n.DataId, n.Group)
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
	// 敏感字段解密：ENCv1 密文 → 内存明文（就地替换，不回写、不落盘、不打日志）
	if err := DecryptSecrets(cipher, &c); err != nil {
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
	// 调试路由已按 profile 白名单收紧（router.debugProfile），此处兜底要求 mode 显式声明，
	// 避免 gin 以默认 debug 模式启动、日志冗余或模式语义混乱
	if c.App.Mode == "" {
		return errors.New("prod 环境 app.mode 缺失：请在 Nacos 的 photography-server-prod.yaml 配置 app.mode: release")
	}
	if c.JWT.Secret == "" || c.JWT.Secret == defaultJWTSecret {
		return errors.New("prod 环境 jwt.secret 缺失或为默认弱值：请在 Nacos 的 photography-server-prod.yaml 配置（推荐 ENCv1: 密文），或临时用 APP_JWT_SECRET 覆盖")
	}
	if c.DB.Host == "" || c.DB.User == "" || c.DB.Password == "" {
		return errors.New("prod 环境 db 连接信息（host/user/password）缺失：请在 Nacos 的 photography-server-prod.yaml 配置（推荐 ENCv1: 密文），或临时用 APP_DB_HOST / APP_DB_USER / APP_DB_PASSWORD 覆盖")
	}
	return nil
}
