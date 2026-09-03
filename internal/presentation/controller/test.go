package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"photography-server/internal/enum"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"photography-server/internal/infrastructure"
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

// NATSPub 发布消息（非持久化）
// POST /test/nats/pub
func (h *Controller) NATSPub(c *gin.Context) {
	var req natsPubReq
	if err := h.bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	if h.Svc.NatsClient() == nil {
		response.Fail(c, errs.Internal("nats 未连接"))
		return
	}
	if err := h.Svc.NatsClient().Publish(req.Subject, []byte(req.Msg)); err != nil {
		response.Fail(c, errs.Internal("nats pub 失败: "+err.Error()))
		return
	}
	response.OK(c, gin.H{"subject": req.Subject, "msg": req.Msg, "mode": "non-persistent"})
}

// NATSPubPersistent 发布消息（持久化，通过 JetStream）
// POST /test/nats/pub-persistent
func (h *Controller) NATSPubPersistent(c *gin.Context) {
	var req natsPubReq
	if err := h.bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	client := h.Svc.NatsClient()
	if client == nil {
		response.Fail(c, errs.Internal("nats 未连接"))
		return
	}
	if !client.IsJetStreamEnabled() {
		response.Fail(c, errs.Internal("jetStream 未启用"))
		return
	}
	subject := "photography." + req.Subject
	ack, err := client.PublishPersistent(subject, []byte(req.Msg))
	if err != nil {
		response.Fail(c, errs.Internal("nats persistent pub 失败: "+err.Error()))
		return
	}
	response.OK(c, gin.H{
		"subject":  subject,
		"msg":      req.Msg,
		"mode":     "persistent",
		"stream":   ack.Stream,
		"sequence": ack.Sequence,
	})
}

// NATSRequest 发布请求并等待回复
// POST /test/nats/request
func (h *Controller) NATSRequest(c *gin.Context) {
	var req natsPubReq
	if err := h.bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	if h.Svc.NatsClient() == nil {
		response.Fail(c, errs.Internal("nats 未连接"))
		return
	}
	data, err := h.Svc.NatsClient().Request(req.Subject, []byte(req.Msg), 3*time.Second)
	if err != nil {
		response.Fail(c, errs.Internal("nats request 超时或失败: "+err.Error()))
		return
	}
	response.OK(c, gin.H{"subject": req.Subject, "reply": string(data)})
}

// NATSStatus 检查 NATS 连接状态
// POST /test/nats/status
func (h *Controller) NATSStatus(c *gin.Context) {
	client := h.Svc.NatsClient()
	if client == nil {
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
		"jetstream": client.IsJetStreamEnabled(),
	})
}

// NATSPubPull 发布消息到 Pull 订阅的 subject
// POST /test/nats/pub-pull
func (h *Controller) NATSPubPull(c *gin.Context) {
	var req natsPubReq
	if err := h.bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	client := h.Svc.NatsClient()
	if client == nil {
		response.Fail(c, errs.Internal("nats 未连接"))
		return
	}
	subject := "photography." + req.Subject
	ack, err := client.PublishPersistent(subject, []byte(req.Msg))
	if err != nil {
		response.Fail(c, errs.Internal("nats pub-pull 失败: "+err.Error()))
		return
	}
	response.OK(c, gin.H{
		"subject":  subject,
		"msg":      req.Msg,
		"mode":     "pull",
		"stream":   ack.Stream,
		"sequence": ack.Sequence,
	})
}

func (h *Controller) Test(c *gin.Context) {
	status := enum.OrderStatusPendingDeposit
	fmt.Println("当前状态:", status)
	statusName := enum.OrderStatusName(status)
	fmt.Println("当前状态名称：", statusName)
}

// ======================== Elasticsearch 示例 ========================

// ESStatus 检查 ES 连接状态
// POST /test/es/status
func (h *Controller) ESStatus(c *gin.Context) {
	if h.Svc.ES() == nil {
		response.Fail(c, errs.Internal("elasticsearch 未连接"))
		return
	}
	res, err := h.Svc.ES().Info()
	if err != nil {
		response.Fail(c, errs.Internal("elasticsearch info 失败: "+err.Error()))
		return
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	var info map[string]interface{}
	json.Unmarshal(body, &info)
	response.OK(c, gin.H{"status": "connected", "info": info})
}

type esIndexReq struct {
	Index string      `json:"index" binding:"required"`
	Body  interface{} `json:"body" binding:"required"`
}

// ESIndex 创建文档
// POST /test/es/index
func (h *Controller) ESIndex(c *gin.Context) {
	if h.Svc.ES() == nil {
		response.Fail(c, errs.Internal("elasticsearch 未连接"))
		return
	}
	var req esIndexReq
	if err := h.bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	data, _ := json.Marshal(req.Body)
	res, err := h.Svc.ES().API.Index(req.Index, bytes.NewReader(data))
	if err != nil {
		response.Fail(c, errs.Internal("es index 失败: "+err.Error()))
		return
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)
	response.OK(c, result)
}

type esSearchReq struct {
	Index  string   `json:"index" binding:"required"`
	Query  string   `json:"query" binding:"required"`
	Fields []string `json:"fields"` // 可选，指定要搜索的字段；为空时使用默认字段列表
}

// ESSearch 搜索文档
// POST /test/es/search
// 注意：Elasticsearch 7.0+ 已移除 _all 字段，不能再对其做 match 查询。
// 默认对 title/content 做多字段匹配，也可通过 fields 参数指定字段。
func (h *Controller) ESSearch(c *gin.Context) {
	if h.Svc.ES() == nil {
		response.Fail(c, errs.Internal("elasticsearch 未连接"))
		return
	}
	var req esSearchReq
	if err := h.bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}

	fields := req.Fields
	if len(fields) == 0 {
		fields = []string{"title", "content"}
	}

	queryBody := map[string]interface{}{
		"query": map[string]interface{}{
			"multi_match": map[string]interface{}{
				"query":  req.Query,
				"fields": fields,
			},
		},
	}
	queryBytes, err := json.Marshal(queryBody)
	if err != nil {
		response.Fail(c, errs.Internal("构造查询失败: "+err.Error()))
		return
	}

	res, err := h.Svc.ES().API.Search(
		h.Svc.ES().API.Search.WithIndex(req.Index),
		h.Svc.ES().API.Search.WithBody(bytes.NewReader(queryBytes)),
	)
	if err != nil {
		response.Fail(c, errs.Internal("es search 失败: "+err.Error()))
		return
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)
	response.OK(c, result)
}

type esListReq struct {
	Index    string `json:"index" binding:"required"`
	Page     int    `json:"page"`      // 页码，从 1 开始，默认 1
	PageSize int    `json:"page_size"` // 每页条数，默认 10
}

// ESList 查询所有文档（match_all）
// POST /test/es/list
func (h *Controller) ESList(c *gin.Context) {
	if h.Svc.ES() == nil {
		response.Fail(c, errs.Internal("elasticsearch 未连接"))
		return
	}
	var req esListReq
	if err := h.bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 10
	}
	// 页码换算成 ES 的 from 偏移量
	from := (req.Page - 1) * req.PageSize

	queryBody := map[string]interface{}{
		"query": map[string]interface{}{
			"match_all": map[string]interface{}{},
		},
		"size": req.PageSize,
		"from": from,
		"sort": []map[string]interface{}{
			{"_score": map[string]interface{}{"order": "desc"}},
		},
	}
	queryBytes, err := json.Marshal(queryBody)
	if err != nil {
		response.Fail(c, errs.Internal("构造查询失败: "+err.Error()))
		return
	}

	res, err := h.Svc.ES().API.Search(
		h.Svc.ES().API.Search.WithIndex(req.Index),
		h.Svc.ES().API.Search.WithBody(bytes.NewReader(queryBytes)),
	)
	if err != nil {
		response.Fail(c, errs.Internal("es list 失败: "+err.Error()))
		return
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)
	response.OK(c, result)
}

type esDeleteReq struct {
	Index string `json:"index" binding:"required"`
	ID    string `json:"id" binding:"required"`
}

// ESDelete 删除文档
// POST /test/es/delete
func (h *Controller) ESDelete(c *gin.Context) {
	if h.Svc.ES() == nil {
		response.Fail(c, errs.Internal("elasticsearch 未连接"))
		return
	}
	var req esDeleteReq
	if err := h.bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	res, err := h.Svc.ES().API.Delete(req.Index, req.ID)
	if err != nil {
		response.Fail(c, errs.Internal("es delete 失败: "+err.Error()))
		return
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)
	response.OK(c, result)
}

// ======================== MongoDB 示例 ========================

// MongoStatus 检查 MongoDB 连接状态
// POST /test/mongo/status
func (h *Controller) MongoStatus(c *gin.Context) {
	if !infrastructure.MongoIsConnected() {
		response.Fail(c, errs.Internal("mongodb 未连接"))
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := h.Svc.Mongo().Ping(ctx, nil); err != nil {
		response.Fail(c, errs.Internal("mongodb ping 失败: "+err.Error()))
		return
	}
	response.OK(c, gin.H{"status": "connected"})
}

type mongoInsertReq struct {
	Collection string                 `json:"collection" binding:"required"`
	Doc        map[string]interface{} `json:"doc" binding:"required"`
}

// MongoInsert 插入单个文档
// POST /test/mongo/insert
func (h *Controller) MongoInsert(c *gin.Context) {
	if !infrastructure.MongoIsConnected() {
		response.Fail(c, errs.Internal("mongodb 未连接"))
		return
	}
	var req mongoInsertReq
	if err := h.bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	res, err := h.Svc.MongoDB().Collection(req.Collection).InsertOne(ctx, req.Doc)
	if err != nil {
		response.Fail(c, errs.Internal("mongo insert 失败: "+err.Error()))
		return
	}
	response.OK(c, gin.H{"inserted_id": res.InsertedID})
}

type mongoInsertManyReq struct {
	Collection string                   `json:"collection" binding:"required"`
	Docs       []map[string]interface{} `json:"docs" binding:"required"`
}

// MongoInsertMany 批量插入文档
// POST /test/mongo/insert-many
func (h *Controller) MongoInsertMany(c *gin.Context) {
	if !infrastructure.MongoIsConnected() {
		response.Fail(c, errs.Internal("mongodb 未连接"))
		return
	}
	var req mongoInsertManyReq
	if err := h.bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	docs := make([]interface{}, 0, len(req.Docs))
	for _, d := range req.Docs {
		docs = append(docs, d)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	res, err := h.Svc.MongoDB().Collection(req.Collection).InsertMany(ctx, docs)
	if err != nil {
		response.Fail(c, errs.Internal("mongo insert-many 失败: "+err.Error()))
		return
	}
	response.OK(c, gin.H{"inserted_count": len(res.InsertedIDs), "inserted_ids": res.InsertedIDs})
}

type mongoFindReq struct {
	Collection string                 `json:"collection" binding:"required"`
	Filter     map[string]interface{} `json:"filter"` // 查询条件，为空查全部
	Page       int                    `json:"page"`
	PageSize   int                    `json:"page_size"`
}

// MongoFind 查询文档（支持分页）
// POST /test/mongo/find
func (h *Controller) MongoFind(c *gin.Context) {
	if !infrastructure.MongoIsConnected() {
		response.Fail(c, errs.Internal("mongodb 未连接"))
		return
	}
	var req mongoFindReq
	if err := h.bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 10
	}
	filter := req.Filter
	if filter == nil {
		filter = bson.M{}
	}
	coll := h.Svc.MongoDB().Collection(req.Collection)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 总数
	total, err := coll.CountDocuments(ctx, filter)
	if err != nil {
		response.Fail(c, errs.Internal("mongo count 失败: "+err.Error()))
		return
	}

	skip := int64((req.Page - 1) * req.PageSize)
	limit := int64(req.PageSize)
	opts := options.Find().SetSkip(skip).SetLimit(limit)
	cursor, err := coll.Find(ctx, filter, opts)
	if err != nil {
		response.Fail(c, errs.Internal("mongo find 失败: "+err.Error()))
		return
	}
	defer cursor.Close(ctx)

	var docs []map[string]interface{}
	if err := cursor.All(ctx, &docs); err != nil {
		response.Fail(c, errs.Internal("mongo decode 失败: "+err.Error()))
		return
	}
	response.OK(c, gin.H{
		"total":     total,
		"page":      req.Page,
		"page_size": req.PageSize,
		"list":      docs,
	})
}

type mongoFindOneReq struct {
	Collection string                 `json:"collection" binding:"required"`
	Filter     map[string]interface{} `json:"filter" binding:"required"`
}

// MongoFindOne 查询单个文档
// POST /test/mongo/find-one
func (h *Controller) MongoFindOne(c *gin.Context) {
	if !infrastructure.MongoIsConnected() {
		response.Fail(c, errs.Internal("mongodb 未连接"))
		return
	}
	var req mongoFindOneReq
	if err := h.bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var doc map[string]interface{}
	err := h.Svc.MongoDB().Collection(req.Collection).FindOne(ctx, req.Filter).Decode(&doc)
	if err != nil {
		response.Fail(c, errs.Internal("mongo find-one 失败: "+err.Error()))
		return
	}
	response.OK(c, doc)
}

type mongoUpdateReq struct {
	Collection string                 `json:"collection" binding:"required"`
	Filter     map[string]interface{} `json:"filter" binding:"required"`
	Update     map[string]interface{} `json:"update" binding:"required"`
}

// MongoUpdate 更新文档（单个）
// POST /test/mongo/update
func (h *Controller) MongoUpdate(c *gin.Context) {
	if !infrastructure.MongoIsConnected() {
		response.Fail(c, errs.Internal("mongodb 未连接"))
		return
	}
	var req mongoUpdateReq
	if err := h.bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	res, err := h.Svc.MongoDB().Collection(req.Collection).UpdateOne(ctx, req.Filter, req.Update)
	if err != nil {
		response.Fail(c, errs.Internal("mongo update 失败: "+err.Error()))
		return
	}
	response.OK(c, gin.H{
		"matched_count":  res.MatchedCount,
		"modified_count": res.ModifiedCount,
		"upserted_id":    res.UpsertedID,
	})
}

type mongoDeleteReq struct {
	Collection string                 `json:"collection" binding:"required"`
	Filter     map[string]interface{} `json:"filter" binding:"required"`
}

// MongoDelete 删除文档（可删多个）
// POST /test/mongo/delete
func (h *Controller) MongoDelete(c *gin.Context) {
	if !infrastructure.MongoIsConnected() {
		response.Fail(c, errs.Internal("mongodb 未连接"))
		return
	}
	var req mongoDeleteReq
	if err := h.bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	res, err := h.Svc.MongoDB().Collection(req.Collection).DeleteMany(ctx, req.Filter)
	if err != nil {
		response.Fail(c, errs.Internal("mongo delete 失败: "+err.Error()))
		return
	}
	response.OK(c, gin.H{"deleted_count": res.DeletedCount})
}

type mongoDeleteIDReq struct {
	Collection string `json:"collection" binding:"required"`
	ID         string `json:"id" binding:"required"`
}

// MongoDeleteByID 按 ObjectID 删除文档
// POST /test/mongo/delete-by-id
func (h *Controller) MongoDeleteByID(c *gin.Context) {
	if !infrastructure.MongoIsConnected() {
		response.Fail(c, errs.Internal("mongodb 未连接"))
		return
	}
	var req mongoDeleteIDReq
	if err := h.bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	oid, err := bson.ObjectIDFromHex(req.ID)
	if err != nil {
		response.Fail(c, errs.BadRequest("无效的 ObjectID: "+err.Error()))
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	res, err := h.Svc.MongoDB().Collection(req.Collection).DeleteOne(ctx, bson.M{"_id": oid})
	if err != nil {
		response.Fail(c, errs.Internal("mongo delete-by-id 失败: "+err.Error()))
		return
	}
	response.OK(c, gin.H{"deleted_count": res.DeletedCount})
}
