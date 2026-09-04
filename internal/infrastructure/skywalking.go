package infrastructure

import (
	"os"
	"sync"
	"time"

	"github.com/SkyAPM/go2sky"
	"github.com/SkyAPM/go2sky/reporter"

	"photography-server/internal/config"
	"photography-server/internal/pkg/logger"
)

// SkyWalking 链路追踪（go2sky SDK，gRPC 异步上报 OAP）。
// Go 无 JVM 字节码增强能力，故采用手动埋点方案：gin 中间件自动产生 entry span，
// 业务内可用 SkyWalking().CreateLocalSpan(ctx) 追加子 span。
// 与其他基础设施一致的容错约定：enable=false 或 OAP 不可达时仅告警，不阻塞服务启动
// （gRPC reporter 为非阻塞拨号，OAP 未启动时 tracer 照常工作，数据在后台重试后丢弃/补报）。

var (
	swTracer   *go2sky.Tracer
	swReporter go2sky.Reporter
	swOnce     sync.Once
	swInitErr  error
)

// InitSkyWalking 初始化 SkyWalking tracer 单例
func InitSkyWalking(c *config.SkyWalking) error {
	swOnce.Do(func() {
		if !c.Enable {
			logger.Infof("skywalking disabled, skipping")
			return
		}
		if c.OapAddr == "" {
			logger.Warnf("skywalking oap_addr is empty, skipping")
			return
		}
		instance := c.Instance
		if instance == "" {
			if host, err := os.Hostname(); err == nil {
				instance = host
			}
		}
		r, err := reporter.NewGRPCReporter(c.OapAddr, reporter.WithCheckInterval(5*time.Second))
		if err != nil {
			swInitErr = err
			logger.Errorf("skywalking reporter init failed: %v", err)
			return
		}
		t, err := go2sky.NewTracer(c.Service, go2sky.WithReporter(r), go2sky.WithInstance(instance))
		if err != nil {
			swInitErr = err
			logger.Errorf("skywalking tracer init failed: %v", err)
			r.Close()
			return
		}
		swReporter = r
		swTracer = t
		logger.Infof("skywalking tracer initialized: service=%s instance=%s oap=%s", c.Service, instance, c.OapAddr)
	})
	return swInitErr
}

// SkyWalking 获取 tracer 单例（未启用时返回 nil）
func SkyWalking() *go2sky.Tracer {
	return swTracer
}

// SkyWalkingEnabled 是否已启用
func SkyWalkingEnabled() bool {
	return swTracer != nil
}

// CloseSkyWalking 关闭 reporter，尽量把缓冲的 span 发完（服务优雅退出时调用）
func CloseSkyWalking() {
	if swReporter != nil {
		swReporter.Close()
	}
}
