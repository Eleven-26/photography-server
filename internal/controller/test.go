package controller

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"

	"photography-server/internal/pkg/errs"
	"photography-server/internal/response"
)

// ======================== Redis 示例 ========================

type redisSetReq struct {
	Key   string `json:"key" binding:"required"`
	Value string `json:"value" binding:"required"`
	TTL   int    `json:"ttl"` // 秒，0=不过期
}

// RedisSet 写入 Redis
// POST /test/redis/set
func (h *Controller) RedisSet(c *gin.Context) {
	var req redisSetReq
	if err := h.bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	if h.Svc.RDB() == nil {
		response.Fail(c, errs.Internal("redis 未连接"))
		return
	}
	ctx := context.Background()
	ttl := time.Duration(req.TTL) * time.Second
	if err := h.Svc.RDB().Set(ctx, req.Key, req.Value, ttl).Err(); err != nil {
		response.Fail(c, errs.Internal("redis set 失败: "+err.Error()))
		return
	}
	response.OK(c, gin.H{"key": req.Key, "value": req.Value, "ttl": req.TTL})
}

type redisGetReq struct {
	Key string `json:"key" binding:"required"`
}

// RedisGet 读取 Redis
// POST /test/redis/get
func (h *Controller) RedisGet(c *gin.Context) {
	var req redisGetReq
	if err := h.bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	if h.Svc.RDB() == nil {
		response.Fail(c, errs.Internal("redis 未连接"))
		return
	}
	ctx := context.Background()
	val, err := h.Svc.RDB().Get(ctx, req.Key).Result()
	if err != nil {
		response.Fail(c, errs.Internal("redis get 失败: "+err.Error()))
		return
	}
	response.OK(c, gin.H{"key": req.Key, "value": val})
}

// RedisDel 删除 Redis key
// POST /test/redis/del
func (h *Controller) RedisDel(c *gin.Context) {
	var req redisGetReq
	if err := h.bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	if h.Svc.RDB() == nil {
		response.Fail(c, errs.Internal("redis 未连接"))
		return
	}
	ctx := context.Background()
	n, err := h.Svc.RDB().Del(ctx, req.Key).Result()
	if err != nil {
		response.Fail(c, errs.Internal("redis del 失败: "+err.Error()))
		return
	}
	response.OK(c, gin.H{"key": req.Key, "deleted": n})
}

// RedisPing 测试 Redis 连通性
// POST /test/redis/ping
func (h *Controller) RedisPing(c *gin.Context) {
	if h.Svc.RDB() == nil {
		response.Fail(c, errs.Internal("redis 未连接"))
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := h.Svc.RDB().Ping(ctx).Err(); err != nil {
		response.Fail(c, errs.Internal("redis ping 失败: "+err.Error()))
		return
	}
	response.OK(c, gin.H{"status": "pong"})
}

// ======================== NATS 示例 ========================

type natsPubReq struct {
	Subject string `json:"subject" binding:"required"`
	Msg     string `json:"msg" binding:"required"`
}

// NATSPub 发布消息
// POST /test/nats/pub
func (h *Controller) NATSPub(c *gin.Context) {
	var req natsPubReq
	if err := h.bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	if h.Svc.NATS() == nil {
		response.Fail(c, errs.Internal("nats 未连接"))
		return
	}
	if err := h.Svc.NATS().Publish(req.Subject, []byte(req.Msg)); err != nil {
		response.Fail(c, errs.Internal("nats pub 失败: "+err.Error()))
		return
	}
	response.OK(c, gin.H{"subject": req.Subject, "msg": req.Msg})
}

// NATSRequest 发布请求并等待回复（demo 用 inbox）
// POST /test/nats/request
func (h *Controller) NATSRequest(c *gin.Context) {
	var req natsPubReq
	if err := h.bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	if h.Svc.NATS() == nil {
		response.Fail(c, errs.Internal("nats 未连接"))
		return
	}
	msg, err := h.Svc.NATS().Request(req.Subject, []byte(req.Msg), 3*time.Second)
	if err != nil {
		response.Fail(c, errs.Internal("nats request 超时或失败: "+err.Error()))
		return
	}
	response.OK(c, gin.H{"subject": req.Subject, "reply": string(msg.Data)})
}

// NATSStatus 检查 NATS 连接状态
// POST /test/nats/status
func (h *Controller) NATSStatus(c *gin.Context) {
	if h.Svc.NATS() == nil {
		response.Fail(c, errs.Internal("nats 未连接"))
		return
	}
	nc := h.Svc.NATS()
	status := "disconnected"
	if nc.IsConnected() {
		status = "connected"
	}
	servers := nc.Servers()
	response.OK(c, gin.H{
		"status":    status,
		"server_id": nc.ConnectedServerId(),
		"servers":   servers,
	})
}
