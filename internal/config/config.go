package config

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	App        App        `mapstructure:"app"`
	JWT        JWT        `mapstructure:"jwt"`
	DB         DB         `mapstructure:"db"`
	Redis      Redis      `mapstructure:"redis"`
	NATS       NATS       `mapstructure:"nats"`
	Mongo      Mongo      `mapstructure:"mongodb"`
	Log        Log        `mapstructure:"log"`
	Upload     Upload     `mapstructure:"upload"`
	XxlJob     XxlJob     `mapstructure:"xxljob"`
	ES         ES         `mapstructure:"elasticsearch"`
	SkyWalking SkyWalking `mapstructure:"skywalking"`
}

type ES struct {
	Enable   bool     `mapstructure:"enable"`
	Urls     []string `mapstructure:"urls"`
	Username string   `mapstructure:"username"`
	Password string   `mapstructure:"password"`
}

// SkyWalking 链路追踪配置：go2sky gRPC 上报 OAP（默认 11800）
type SkyWalking struct {
	Enable   bool   `mapstructure:"enable"`   // 总开关；false 时不创建 tracer，中间件跳过挂载
	OapAddr  string `mapstructure:"oap_addr"` // OAP gRPC 地址，如 127.0.0.1:11800
	Service  string `mapstructure:"service"`  // SkyWalking 中的服务名
	Instance string `mapstructure:"instance"` // 实例名，留空默认取主机名
}

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

// Load 加载配置：基础配置 config.yaml + 环境覆盖 config.<profile>.yaml + APP_* 环境变量
// 优先级：APP_* 环境变量 > config.<profile>.yaml > config.yaml（viper AutomaticEnv）
// 注意：不做 ${VAR} 模板展开；Unmarshal 只能覆盖配置文件中已存在的 key，新增配置项需同步维护各 yaml。
func Load(basePath, profile string) (*Config, error) {
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
