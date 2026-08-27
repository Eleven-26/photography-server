package config

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	App    App    `mapstructure:"app"`
	JWT    JWT    `mapstructure:"jwt"`
	DB     DB     `mapstructure:"db"`
	Log    Log    `mapstructure:"log"`
	Upload Upload `mapstructure:"upload"`
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

type Log struct {
	Level string `mapstructure:"level"`
}

type Upload struct {
	Dir       string `mapstructure:"dir"`
	MaxSizeMB int    `mapstructure:"max_size_mb"`
}

// Load 加载配置：基础配置 config.yaml + 环境覆盖 config.<profile>.yaml + APP_* 环境变量
// 配置文件中的 ${VAR} 占位符会自动用同名环境变量展开。
func Load(basePath, profile string) (*Config, error) {
	if profile == "" {
		profile = "dev"
	}
	profile = strings.ToLower(profile)

	v := viper.New()
	v.SetConfigType("yaml")
	if err := readConfigFile(v, basePath, false); err != nil {
		return nil, err
	}

	// 环境专用配置覆盖基础配置
	ppath := profilePath(basePath, profile)
	if _, err := os.Stat(ppath); err == nil {
		if err := readConfigFile(v, ppath, true); err != nil {
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
	return &c, nil
}

// readConfigFile 读取配置文件并对 ${VAR} 占位符做环境变量展开；merge=true 时合并进已有配置
func readConfigFile(v *viper.Viper, path string, merge bool) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	expanded := os.Expand(string(data), os.Getenv)
	if merge {
		return v.MergeConfig(strings.NewReader(expanded))
	}
	return v.ReadConfig(strings.NewReader(expanded))
}

// profilePath 计算环境配置文件名：config/config.yaml + prod => config/config.prod.yaml
func profilePath(base, profile string) string {
	dir := filepath.Dir(base)
	name := filepath.Base(base)
	ext := filepath.Ext(name)
	stem := strings.TrimSuffix(name, ext)
	return filepath.Join(dir, stem+"."+profile+ext)
}
