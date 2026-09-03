package infrastructure

import (
	"context"
	"strings"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"photography-server/internal/config"
	"photography-server/internal/pkg/logger"
)

var (
	mongoClient *mongo.Client
	mongoDB     *mongo.Database
	mongoOnce   sync.Once
	mongoErr    error
)

// InitMongoDB 初始化 MongoDB 单例
func InitMongoDB(c *config.Mongo) error {
	mongoOnce.Do(func() {
		if !c.Enable {
			logger.Infof("mongodb disabled, skipping")
			return
		}
		if strings.TrimSpace(c.URI) == "" {
			logger.Warnf("mongodb uri is empty, skipping")
			return
		}

		// 若配置了 username/password 但 uri 未带认证信息，则补全认证
		uri := c.URI
		if c.Username != "" && c.Password != "" {
			uri = buildMongoURI(uri, c.Username, c.Password)
		}

		clientOpts := options.Client().ApplyURI(uri)
		client, err := mongo.Connect(clientOpts)
		if err != nil {
			logger.Errorf("mongodb connect failed: %v", err)
			mongoErr = err
			return
		}

		// 验证连接
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := client.Ping(ctx, nil); err != nil {
			logger.Errorf("mongodb ping failed: %v", err)
			mongoErr = err
			return
		}

		mongoClient = client
		if c.Database != "" {
			mongoDB = client.Database(c.Database)
		}
		logger.Infof("mongodb connected: %s", c.URI)
	})
	return mongoErr
}

// buildMongoURI 在 uri 中注入用户名密码（若尚未包含认证信息）
func buildMongoURI(uri, username, password string) string {
	// 已包含认证信息则直接返回
	if strings.Contains(uri, "@") {
		return uri
	}
	prefix := "mongodb://"
	if strings.HasPrefix(uri, "mongodb+srv://") {
		prefix = "mongodb+srv://"
	} else if strings.HasPrefix(uri, "mongodb://") {
		prefix = "mongodb://"
	}
	rest := strings.TrimPrefix(uri, prefix)
	return prefix + username + ":" + password + "@" + rest
}

// Mongo 获取 MongoDB 客户端单例
func Mongo() *mongo.Client {
	return mongoClient
}

// MongoDatabase 获取默认数据库
func MongoDatabase() *mongo.Database {
	return mongoDB
}

// MongoIsConnected 是否已连接
func MongoIsConnected() bool {
	return mongoClient != nil
}

// MongoDisconnect 关闭连接（服务优雅退出时调用）
func MongoDisconnect(ctx context.Context) error {
	if mongoClient == nil {
		return nil
	}
	return mongoClient.Disconnect(ctx)
}
