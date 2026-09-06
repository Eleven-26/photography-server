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

// Jaeger 链路通道（OpenTelemetry SDK 实现，OTLP gRPC 上报 Jaeger v2，存储 ClickHouse）。
// 数据流：应用 --OTLP:4317--> jaeger(collector+query 一体) --> ClickHouse，Jaeger UI(:16686) 按 trace_id 查。
// 采用手动埋点方案：gin 中间件(otelgin)自动产生 entry span，SQL 由 gorm OTel 插件产生 client span，
// xxl-job / NATS 入口在各自 wrapper 里创建/续接 span，业务内可用 JaegerTracer().Start(ctx, "op") 追加子 span。
// 与其他基础设施一致的容错约定：enable=false 或 endpoint 为空时跳过；
// OTLP 为异步批量上报，Jaeger 不可达不影响请求路径，数据在后台重试后丢弃。
// 上报链路端到端为 W3C TraceContext 传播（otel 默认），NATS 消息头透传 traceparent 依赖本通道的全局 propagator。
// 注意：SkyWalking-go（native）通道与本节无关 —— 它由编译期注入的 agent 自动埋点、直连 OAP:11800，
// 无运行时开关；两通道各自独立但勿同时开启（同一请求会双 span/双上报）。

var (
	jaegerTracerProvider *sdktrace.TracerProvider
	jaegerTracer         trace.Tracer
	jaegerOnce           sync.Once
	jaegerInitErr        error
)

// InitJaeger 初始化 Jaeger 通道的 tracer provider 单例
func InitJaeger(c *config.Jaeger) error {
	jaegerOnce.Do(func() {
		if !c.Enable {
			logger.Infof("jaeger disabled, skipping")
			return
		}
		if c.Endpoint == "" {
			logger.Warnf("jaeger endpoint is empty, skipping")
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
			jaegerInitErr = err
			logger.Errorf("jaeger exporter init failed: %v", err)
			return
		}
		res, err := resource.New(ctx, resource.WithAttributes(
			attribute.String("service.name", c.Service),
			attribute.String("service.instance.id", instance),
		))
		if err != nil {
			jaegerInitErr = err
			logger.Errorf("jaeger resource init failed: %v", err)
			return
		}
		jaegerTracerProvider = sdktrace.NewTracerProvider(
			sdktrace.WithResource(res),
			sdktrace.WithBatcher(exp, sdktrace.WithBatchTimeout(5*time.Second)),
		)
		otel.SetTracerProvider(jaegerTracerProvider)
		otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{}, propagation.Baggage{},
		))
		jaegerTracer = jaegerTracerProvider.Tracer(c.Service)
		logger.Infof("jaeger initialized: service=%s instance=%s endpoint=%s", c.Service, instance, c.Endpoint)
	})
	return jaegerInitErr
}

// JaegerTracer Jaeger 通道的 tracer（未启用时返回 nil，业务侧判空跳过埋点）
func JaegerTracer() trace.Tracer {
	return jaegerTracer
}

// JaegerEnabled Jaeger 通道是否已启用
func JaegerEnabled() bool {
	return jaegerTracer != nil
}

// CloseJaeger 刷出缓冲中的 span 并释放资源（服务优雅退出时调用）
func CloseJaeger(ctx context.Context) {
	if jaegerTracerProvider != nil {
		if err := jaegerTracerProvider.Shutdown(ctx); err != nil {
			logger.Errorf("jaeger shutdown error: %v", err)
		}
	}
}
