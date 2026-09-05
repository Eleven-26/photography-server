package infrastructure

import (
	"context"
	"os"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"

	"photography-server/internal/config"
	"photography-server/internal/pkg/logger"
)

// SkyWalking 链路追踪方案（OpenTelemetry SDK 实现，OTLP gRPC 上报 otel-collector）。
// 数据流：应用 --OTLP:4317--> otel-collector --OTLP:11800--> SkyWalking OAP --> BanyanDB。
// Go 无 JVM 字节码增强能力，故采用手动埋点方案：gin 中间件(otelgin)自动产生 entry span，
// 业务内可用 SkyWalkingTracer().Start(ctx, "op") 追加子 span。
// 与其他基础设施一致的容错约定：enable=false 或 endpoint 为空时跳过；
// OTLP 为异步批量上报，collector/OAP 不可达不影响请求路径，数据在后台重试后丢弃。
// 上报链路端到端必须为 W3C TraceContext 传播（otel 默认），与 SkyWalking 兼容。
// 命名说明：配置段/API 以方案名（skywalking）命名，与后续新增的其他链路追踪方案
// （如 otel+Jaeger，独立配置段 jaeger:）相互区分。

var (
	tracerProvider *sdktrace.TracerProvider
	appTracer      trace.Tracer
	swOnce         sync.Once
	swInitErr      error
)

// InitSkyWalking 初始化 SkyWalking 方案的 tracer provider 单例
func InitSkyWalking(c *config.SkyWalking) error {
	swOnce.Do(func() {
		if !c.Enable {
			logger.Infof("skywalking disabled, skipping")
			return
		}
		if c.Endpoint == "" {
			logger.Warnf("skywalking endpoint is empty, skipping")
			return
		}
		instance := c.Instance
		if instance == "" {
			if host, err := os.Hostname(); err == nil {
				instance = host
			}
		}
		ctx := context.Background()
		exp, err := otlptracegrpc.New(ctx,
			otlptracegrpc.WithEndpoint(c.Endpoint),
			otlptracegrpc.WithInsecure(),
		)
		if err != nil {
			swInitErr = err
			logger.Errorf("skywalking exporter init failed: %v", err)
			return
		}
		res, err := resource.New(ctx, resource.WithAttributes(
			attribute.String("service.name", c.Service),
			attribute.String("service.instance.id", instance),
		))
		if err != nil {
			swInitErr = err
			logger.Errorf("skywalking resource init failed: %v", err)
			return
		}
		tracerProvider = sdktrace.NewTracerProvider(
			sdktrace.WithResource(res),
			sdktrace.WithBatcher(exp, sdktrace.WithBatchTimeout(5*time.Second)),
		)
		otel.SetTracerProvider(tracerProvider)
		otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{}, propagation.Baggage{},
		))
		appTracer = tracerProvider.Tracer(c.Service)
		logger.Infof("skywalking initialized: service=%s instance=%s endpoint=%s", c.Service, instance, c.Endpoint)
	})
	return swInitErr
}

// SkyWalkingTracer SkyWalking 方案的 tracer（未启用时返回 nil，业务侧判空跳过埋点）
func SkyWalkingTracer() trace.Tracer {
	return appTracer
}

// SkyWalkingEnabled SkyWalking 方案是否已启用
func SkyWalkingEnabled() bool {
	return appTracer != nil
}

// CloseSkyWalking 刷出缓冲中的 span 并释放资源（服务优雅退出时调用）
func CloseSkyWalking(ctx context.Context) {
	if tracerProvider != nil {
		if err := tracerProvider.Shutdown(ctx); err != nil {
			logger.Errorf("skywalking shutdown error: %v", err)
		}
	}
}
