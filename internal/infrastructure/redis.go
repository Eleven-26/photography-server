package infrastructure

import (
	"context"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"

	"photography-server/internal/config"
)

var (
	rdbInstance *redis.Client
	rdbOnce     sync.Once
	rdbErr      error
)

// InitRedis 初始化 Redis 单例
func InitRedis(c *config.Redis) error {
	rdbOnce.Do(func() {
		rdbInstance = redis.NewClient(&redis.Options{
			Addr:     c.Addr,
			Password: c.Password,
			DB:       c.DB,
		})
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		rdbErr = rdbInstance.Ping(ctx).Err()
	})
	return rdbErr
}

// Redis 获取 Redis 单例
func Redis() *redis.Client {
	return rdbInstance
}
