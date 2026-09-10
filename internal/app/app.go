// Package app 定义应用依赖容器，是组合根（cmd/server/main.go）装配依赖的载体。
//
// 分层意义（#40）：业务包（repository / service / middleware / job / mq / presentation）
// 一律不 import infrastructure，只依赖本包暴露的 App（或更窄的注入项）。
// 因此「基础设施实现」与「业务代码」之间只通过 App 的字段类型发生耦合，
// 单测可直接构造 App{DB: 内存库, ...} 而无需启动真实基础设施。
package app

import (
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/nats-io/nats.go"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/gorm"

	"photography-server/internal/config"
	"photography-server/internal/infrastructure"
)

// App 显式持有全部已初始化的基础设施句柄。
//
// 背景（#40）：改造前 repository / service / middleware / presentation 各自直接调用
// infrastructure.MySQL() / Redis() / JaegerTracer() 等包级单例，形成四层全局耦合，
// 后果是无法依赖注入（替换实现只能改源码）、单测被迫依赖真实全局状态、
// 初始化时序隐式耦合（例如必须先 InitJaeger 再 InitMySQL，否则 SQL span 全丢）却无任何约束。
//
// 改造后的约定：
//   - 单例（连接池在本进程中天然唯一）只允许在组合根 cmd/server/main.go 读写一次；
//   - main 用 New 组装出 App，再按需注入各层；
//   - 可选依赖（NATS / ES / Mongo / Tracer）未启用时为 nil，调用方按既有约定判空降级。
type App struct {
	Cfg     *config.Config
	DB      *gorm.DB
	Redis   *redis.Client
	NATS    *nats.Conn
	NatsCli *infrastructure.NatsClient
	ES      *elasticsearch.Client
	Mongo   *mongo.Client
	MongoDB *mongo.Database
	Tracer  trace.Tracer
}

// New 从已初始化的句柄组装容器。仅由组合根调用，使单例的读写都收敛在 main 一处。
func New(cfg *config.Config) *App {
	return &App{
		Cfg:     cfg,
		DB:      infrastructure.MySQL(),
		Redis:   infrastructure.Redis(),
		NATS:    infrastructure.NATS(),
		NatsCli: infrastructure.GetNatsClient(),
		ES:      infrastructure.ES(),
		Mongo:   infrastructure.Mongo(),
		MongoDB: infrastructure.MongoDatabase(),
		Tracer:  infrastructure.JaegerTracer(),
	}
}

// JaegerEnabled 链路（Jaeger/OTel 通道）是否启用。
// 替代原先散落的 infrastructure.JaegerEnabled() 全局判断。
func (a *App) JaegerEnabled() bool {
	return a != nil && a.Tracer != nil
}

// MongoConnected Mongo 是否可用
func (a *App) MongoConnected() bool {
	return a != nil && a.Mongo != nil
}
