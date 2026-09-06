package infrastructure

import (
	"errors"
	"fmt"
	"net"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/config_client"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	nacosutil "github.com/nacos-group/nacos-sdk-go/v2/util"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"

	"photography-server/internal/config"
	"photography-server/internal/pkg/logger"
)

// nacosHub 保存 Nacos 客户端单例与本次注册的实例参数。
// 反注册必须与注册时传同样的 ip/port/group/cluster，否则 Nacos 侧匹配不到实例，所以注册参数在这里留档。
type nacosHub struct {
	once       sync.Once
	conf       config_client.IConfigClient
	naming     naming_client.INamingClient
	cfg        config.Nacos
	registered *vo.RegisterInstanceParam
	err        error
}

var nacosState nacosHub

// NacosEnabled 当前是否已建立 Nacos 客户端（nacos.enable=true 且初始化成功）
func NacosEnabled() bool { return nacosState.naming != nil || nacosState.conf != nil }

// InitNacos 建立配置客户端与命名客户端（幂等：重复调用返回首次结果）。
// 未启用（nacos.enable=false）时不要调用本函数，各查询函数会返回 nil 让业务流程跳过。
func InitNacos(n config.Nacos) error {
	nacosState.once.Do(func() { nacosState.init(n) })
	return nacosState.err
}

func (s *nacosHub) init(n config.Nacos) {
	if n.ServerAddr == "" {
		s.err = errors.New("nacos.server_addr 不能为空（格式 host:8848）")
		return
	}
	host, portStr, err := net.SplitHostPort(n.ServerAddr)
	if err != nil {
		s.err = fmt.Errorf("nacos.server_addr 格式应为 host:port：%w", err)
		return
	}
	port, err := strconv.ParseUint(portStr, 10, 64)
	if err != nil {
		s.err = fmt.Errorf("nacos.server_addr 端口非法：%w", err)
		return
	}

	// v2 SDK 走 gRPC：未显式指定 GrpcPort 时自动用 Port+1000（8848 → 9848）
	servers := []constant.ServerConfig{{IpAddr: host, Port: port, Scheme: "http"}}
	clientCfg := constant.NewClientConfig(
		constant.WithNamespaceId(n.Namespace),
		constant.WithTimeoutMs(n.TimeoutMs),
		constant.WithCacheDir(n.CacheDir),
		constant.WithLogDir(filepath.Join(filepath.Dir(n.CacheDir), "log")),
		constant.WithLogLevel(n.LogLevel),
		constant.WithUsername(n.Username),
		constant.WithPassword(n.Password),
		constant.WithUpdateCacheWhenEmpty(true), // 远端配置被清空时也同步本地快照，避免重启后仍用旧值
	)

	conf, err := clients.NewConfigClient(vo.NacosClientParam{ClientConfig: clientCfg, ServerConfigs: servers})
	if err != nil {
		s.err = fmt.Errorf("创建 Nacos 配置客户端失败：%w", err)
		return
	}
	naming, err := clients.NewNamingClient(vo.NacosClientParam{ClientConfig: clientCfg, ServerConfigs: servers})
	if err != nil {
		s.err = fmt.Errorf("创建 Nacos 命名客户端失败：%w", err)
		return
	}
	s.conf, s.naming, s.cfg = conf, naming, n
	logger.Infof("nacos 客户端就绪: %s (group=%s)", n.ServerAddr, n.Group)
}

// FetchConfig 拉取远端配置内容（YAML 文本），供 config.LoadWithFetcher 在启动阶段调用。
// Nacos 不可达时 SDK 会降级读取本地快照（cache_dir），快照也没有才返回错误（此时启动应终止）。
func FetchConfig(n config.Nacos) (string, error) {
	if err := InitNacos(n); err != nil {
		return "", err
	}
	if nacosState.conf == nil {
		return "", errors.New("nacos 配置客户端未初始化")
	}
	content, err := nacosState.conf.GetConfig(vo.ConfigParam{DataId: n.DataId, Group: n.Group})
	if err != nil {
		return "", err
	}
	if content == "" {
		return "", fmt.Errorf("远端配置为空（data_id=%s group=%s），请检查 Nacos 上是否已发布该配置", n.DataId, n.Group)
	}
	return content, nil
}

// RegisterService 把本实例注册到 Nacos（临时实例：Ephemeral=true，SDK 按 BeatInterval 自动心跳）。
// 进程被 kill 或优雅退出后心跳停止，Nacos 会在约 15s 后摘除实例；优雅退出建议显式调用 DeregisterService。
// appName 作为服务名兜底（nacos.service_name 为空时取它），port 为 HTTP 监听端口。
func RegisterService(appName string, port int, extraMeta map[string]string) (bool, error) {
	if err := InitNacos(nacosState.cfg); err != nil {
		return false, err
	}
	if nacosState.naming == nil {
		return false, errors.New("nacos 命名客户端未初始化")
	}

	name := nacosState.cfg.ServiceName
	if name == "" {
		name = appName
	}
	ip := nacosState.cfg.RegisterIp
	if ip == "" {
		ip = nacosutil.LocalIP()
	}
	if ip == "" {
		ip = "127.0.0.1"
	}

	meta := map[string]string{
		"started_at": time.Now().Format(time.RFC3339),
		"port":       strconv.Itoa(port),
	}
	for k, v := range extraMeta {
		meta[k] = v
	}

	param := vo.RegisterInstanceParam{
		Ip:          ip,
		Port:        uint64(port),
		ServiceName: name,
		GroupName:   nacosState.cfg.Group,
		ClusterName: nacosState.cfg.ClusterName,
		Weight:      nacosState.cfg.Weight,
		Enable:      true,
		Healthy:     true,
		Ephemeral:   true, // 临时实例：靠心跳保活，进程退出自动摘除
		Metadata:    meta,
	}
	ok, err := nacosState.naming.RegisterInstance(param)
	if err != nil {
		return false, fmt.Errorf("注册服务 %s(%s:%d) 失败：%w", name, ip, port, err)
	}
	nacosState.registered = &param
	logger.Infof("已注册到 nacos: %s %s:%d (group=%s cluster=%s)", name, ip, port, param.GroupName, param.ClusterName)
	return ok, nil
}

// DeregisterService 优雅退出时主动摘除实例（用注册时留档的参数，保证 ip/port/group/cluster 一致）
func DeregisterService() (bool, error) {
	if nacosState.naming == nil || nacosState.registered == nil {
		return false, nil
	}
	p := nacosState.registered
	ok, err := nacosState.naming.DeregisterInstance(vo.DeregisterInstanceParam{
		Ip:          p.Ip,
		Port:        p.Port,
		ServiceName: p.ServiceName,
		GroupName:   p.GroupName,
		Cluster:     p.ClusterName,
		Ephemeral:   p.Ephemeral,
	})
	if err != nil {
		return false, fmt.Errorf("反注册服务 %s(%s:%d) 失败：%w", p.ServiceName, p.Ip, p.Port, err)
	}
	nacosState.registered = nil
	logger.Infof("已从 nacos 摘除实例: %s %s:%d", p.ServiceName, p.Ip, p.Port)
	return ok, nil
}

// CloseNacos 关闭客户端（先尝试反注册，避免短暂停留的脏实例）
func CloseNacos() {
	if !NacosEnabled() {
		return
	}
	if _, err := DeregisterService(); err != nil {
		logger.Warnf("nacos 反注册失败: %v", err)
	}
	if nacosState.naming != nil {
		nacosState.naming.CloseClient()
	}
	if nacosState.conf != nil {
		nacosState.conf.CloseClient()
	}
	logger.Infof("nacos 客户端已关闭")
}

// NacosConfig 返回配置段（供 main 判断是否启用、拿 dataId 等）
func NacosConfig() config.Nacos { return nacosState.cfg }
