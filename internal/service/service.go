package service

import (
	"fmt"
	"math"
	"math/rand"
	"time"

	"gorm.io/gorm"
)

// Operator 当前操作人（由认证中间件注入）
type Operator struct {
	UserID    int64
	Username  string
	Nickname  string
	CompanyID int64
	StoreID   int64
	RoleID    int64
}

// Service 业务服务根结构，按领域拆分到不同文件
type Service struct {
	DB        *gorm.DB
	UploadDir string
}

func New(db *gorm.DB, uploadDir string) *Service {
	return &Service{DB: db, UploadDir: uploadDir}
}

// tenant 返回按 company_id 过滤的查询会话，实现 SaaS 多租户隔离
func (s *Service) tenant(op Operator) *gorm.DB {
	return s.DB.Where("company_id = ?", op.CompanyID)
}

func round2(f float64) float64 {
	return math.Round(f*100) / 100
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func orDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}

func orDefaultInt64(v, def int64) int64 {
	if v == 0 {
		return def
	}
	return v
}

// genCode 生成业务编号，如 SL-260817-1234
func genCode(prefix string) string {
	return fmt.Sprintf("%s-%s-%04d", prefix, time.Now().Format("060102"), rand.Intn(10000))
}

// refundRatio 根据拍摄时间距现在的时长计算退款比例
// 规则：>=72小时 全额退款；>=48小时 退80%；>=24小时 退50%；<24小时 不退
func refundRatio(shootDate string, hoursBeforeShoot time.Duration) (float64, string) {
	switch {
	case hoursBeforeShoot >= 72*time.Hour:
		return 1.0, "shoot_gt_72h"
	case hoursBeforeShoot >= 48*time.Hour:
		return 0.8, "shoot_48_72h"
	case hoursBeforeShoot >= 24*time.Hour:
		return 0.5, "shoot_24_48h"
	default:
		return 0, "shoot_lt_24h"
	}
}
