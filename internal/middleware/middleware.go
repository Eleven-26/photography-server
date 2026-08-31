package middleware

import (
	"sync"

	"github.com/gin-gonic/gin"

	"photography-server/internal/config"
	"photography-server/internal/service"
)

type Middlewares struct {
	Cfg *config.Config
}

var (
	mwInstance *Middlewares
	mwOnce     sync.Once
)

// Init 初始化中间件单例
func Init(cfg *config.Config) {
	mwOnce.Do(func() {
		mwInstance = &Middlewares{Cfg: cfg}
	})
}

// Get 获取中间件单例
func Get() *Middlewares {
	return mwInstance
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
