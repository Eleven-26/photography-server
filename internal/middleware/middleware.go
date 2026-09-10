package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/gorm"

	"photography-server/internal/config"
	"photography-server/internal/service"
)

// Deps 中间件所需的依赖，由组合根（main）注入（#40）。
// 改造前中间件直接调用 infrastructure.MySQL()/Redis()/JaegerEnabled() 全局单例。
type Deps struct {
	Cfg    *config.Config
	DB     *gorm.DB
	Redis  *redis.Client
	Tracer trace.Tracer // nil = 链路未启用
}

// Middlewares 中间件集合：显式持有注入的连接句柄，不读取任何包级单例。
type Middlewares struct {
	Cfg    *config.Config
	DB     *gorm.DB
	Redis  *redis.Client
	Tracer trace.Tracer
}

// New 构造中间件集合。
// 替代原 Init/Get 单例：原实现由 router.New 与 main 各调用一次、依赖 sync.Once 兜底，
// 初始化时序隐式耦合；现在由组合根构造一次再显式传入 router。
func New(deps Deps) *Middlewares {
	return &Middlewares{
		Cfg:    deps.Cfg,
		DB:     deps.DB,
		Redis:  deps.Redis,
		Tracer: deps.Tracer,
	}
}

type ctxKey string

// OperatorKey 操作人上下文键，同时用于 gin.Context 与 request context
const OperatorKey ctxKey = "photography.operator"

// GetOperator 从 gin 上下文获取当前操作人
func GetOperator(c *gin.Context) service.Operator {
	v, _ := c.Get(string(OperatorKey))
	op, _ := v.(service.Operator)
	return op
}
