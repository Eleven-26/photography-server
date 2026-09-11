package authcache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// 认证画像缓存（#30）：JWT 已携带用户身份，Auth/StaffAuth/CustomerAuth 每请求回源 DB
// 只为校验状态 + 取画像字段（用户名/角色/门店等）。这里把画像缓存到 Redis（短 TTL），
// 把用户表从每请求热点降下来，同时避免用户表成为登录链路的单点。
//
// 一致性策略：
//   - 短 TTL（60s）：变更最长 60s 自然生效；
//   - 主动失效：停用/改密/改角色等变更点调用 Del* 即时失效（见各变更 service），
//     使"停用账号"立即生效而不是等缓存过期；
//   - fail-open：缓存读取出错（Redis 抖动）时调用方回源 DB，正确性优先于性能，
//     不会因缓存故障误放行/误拦截。

const ttl = 60 * time.Second

// StaffProfile 员工认证所需最小画像（对应 sys_user 行）
type StaffProfile struct {
	Status    int    `json:"status"`
	Username  string `json:"username"`
	Nickname  string `json:"nickname"`
	CompanyID int64  `json:"company_id"`
	StoreID   int64  `json:"store_id"`
	RoleID    int64  `json:"role_id"`
}

// CustomerProfile 客户认证所需最小画像（对应 crm_customer 行）
type CustomerProfile struct {
	Status int    `json:"status"`
	Name   string `json:"name"`
	Mobile string `json:"mobile"`
}

func StaffKey(userID int64) string {
	return fmt.Sprintf("auth:staff:%d", userID)
}

func CustomerKey(companyID, customerID int64) string {
	return fmt.Sprintf("auth:customer:%d:%d", companyID, customerID)
}

// GetStaff 读员工画像缓存；hit=false 表示未命中（可回源 DB）
func GetStaff(ctx context.Context, rdb *redis.Client, userID int64) (p *StaffProfile, hit bool, err error) {
	b, err := rdb.Get(ctx, StaffKey(userID)).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	if err := json.Unmarshal(b, &p); err != nil {
		return nil, false, err
	}
	return p, true, nil
}

// SetStaff 写员工画像缓存（序列化失败仅丢缓存，不影响主流程）
func SetStaff(ctx context.Context, rdb *redis.Client, userID int64, p *StaffProfile) {
	b, err := json.Marshal(p)
	if err != nil {
		return
	}
	rdb.Set(ctx, StaffKey(userID), b, ttl)
}

// DelStaff 删除员工画像缓存（停用/改密/改角色时调用，即时生效）
func DelStaff(ctx context.Context, rdb *redis.Client, userID int64) {
	rdb.Del(ctx, StaffKey(userID))
}

// GetCustomer 读客户画像缓存；hit=false 表示未命中（可回源 DB）
func GetCustomer(ctx context.Context, rdb *redis.Client, companyID, customerID int64) (p *CustomerProfile, hit bool, err error) {
	b, err := rdb.Get(ctx, CustomerKey(companyID, customerID)).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	if err := json.Unmarshal(b, &p); err != nil {
		return nil, false, err
	}
	return p, true, nil
}

// SetCustomer 写客户画像缓存
func SetCustomer(ctx context.Context, rdb *redis.Client, companyID, customerID int64, p *CustomerProfile) {
	b, err := json.Marshal(p)
	if err != nil {
		return
	}
	rdb.Set(ctx, CustomerKey(companyID, customerID), b, ttl)
}

// DelCustomer 删除客户画像缓存（客户停用/拉黑时调用，即时生效）
func DelCustomer(ctx context.Context, rdb *redis.Client, companyID, customerID int64) {
	rdb.Del(ctx, CustomerKey(companyID, customerID))
}

// ---- 角色授权缓存（RBAC）----
//
// 认证阶段需要角色的权限点集合与数据范围。这两项变更频率极低（只在角色配置界面改动），
// 但每请求都要用，故单独缓存且 TTL 远长于画像（画像 60s 为的是"停用即时生效"，
// 授权变更则由保存动作主动失效，无需靠短 TTL 兜底）。
//
// 键设计 perm:role:{companyID}:{roleID} —— 按角色缓存而非按用户，
// 同角色 N 个成员共享一份，成员改角色时自然切换（画像缓存失效已在 UpdateUser 处理）。

// roleTTL 角色授权缓存有效期
const roleTTL = 30 * time.Minute

// RoleAuth 角色授权信息（认证阶段判定所需最小集合）
type RoleAuth struct {
	RoleCode    string   `json:"role_code"`   // 角色编码，admin 短路放行全部权限
	DataScope   int      `json:"data_scope"`  // 数据范围 1-全部 2-本门店 3-仅本人
	Permissions []string `json:"permissions"` // 权限点集合，如 ["order:view","order:create"]
}

func RoleKey(companyID, roleID int64) string {
	return fmt.Sprintf("perm:role:%d:%d", companyID, roleID)
}

// GetRoleAuth 读角色授权缓存；hit=false 表示未命中（可回源 DB）
func GetRoleAuth(ctx context.Context, rdb *redis.Client, companyID, roleID int64) (a *RoleAuth, hit bool, err error) {
	b, err := rdb.Get(ctx, RoleKey(companyID, roleID)).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	if err := json.Unmarshal(b, &a); err != nil {
		return nil, false, err
	}
	return a, true, nil
}

// SetRoleAuth 写角色授权缓存（序列化失败仅丢缓存，不影响主流程）
func SetRoleAuth(ctx context.Context, rdb *redis.Client, companyID, roleID int64, a *RoleAuth) {
	b, err := json.Marshal(a)
	if err != nil {
		return
	}
	rdb.Set(ctx, RoleKey(companyID, roleID), b, roleTTL)
}

// DelRoleAuth 删除角色授权缓存（角色权限/数据范围变更后调用，即时生效）
func DelRoleAuth(ctx context.Context, rdb *redis.Client, companyID, roleID int64) {
	rdb.Del(ctx, RoleKey(companyID, roleID))
}
