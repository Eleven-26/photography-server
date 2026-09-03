package infrastructure

import (
	"strings"
	"sync"

	"github.com/elastic/go-elasticsearch/v8"

	"photography-server/internal/config"
	"photography-server/internal/pkg/logger"
)

var (
	esClient *elasticsearch.Client
	esOnce   sync.Once
)

// InitES 初始化 Elasticsearch 单例
func InitES(c *config.ES) error {
	esOnce.Do(func() {
		if !c.Enable {
			logger.Infof("elasticsearch disabled, skipping")
			return
		}
		if len(c.Urls) == 0 {
			logger.Warnf("elasticsearch urls is empty, skipping")
			return
		}
		cfg := elasticsearch.Config{
			Addresses: c.Urls,
		}
		if c.Username != "" {
			cfg.Username = c.Username
		}
		if c.Password != "" {
			cfg.Password = c.Password
		}
		var err error
		esClient, err = elasticsearch.NewClient(cfg)
		if err != nil {
			logger.Errorf("elasticsearch init failed: %v", err)
			return
		}
		// 验证连接
		res, err := esClient.Info()
		if err != nil {
			logger.Errorf("elasticsearch connect failed: %v", err)
			esClient = nil
			return
		}
		defer res.Body.Close()
		logger.Infof("elasticsearch connected: %s", strings.Join(c.Urls, ","))
	})
	return nil
}

// ES 获取 Elasticsearch 客户端单例
func ES() *elasticsearch.Client {
	return esClient
}

// ESIsConnected 是否已连接
func ESIsConnected() bool {
	return esClient != nil
}

// ErrESNotConnected ES 未连接错误
var ErrESNotConnected = &ESError{Msg: "elasticsearch not connected"}

type ESError struct {
	Msg string
}

func (e *ESError) Error() string {
	return e.Msg
}
