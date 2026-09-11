package service

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"photography-server/internal/domain"
	"photography-server/internal/model"
	"photography-server/internal/pkg/errs"
	"photography-server/internal/repository"
)

// Operator 当前操作人（由认证中间件注入）
// 重构 #32：类型别名，实际定义在 domain 包
type Operator = domain.Operator

// Service 业务服务根结构，按领域拆分到不同文件。
// 分层纪律：service 只依赖 repository（数据访问唯一入口）与组合根注入的依赖，
// 不再读取任何 infrastructure 包级单例（#40）；开启事务统一走 repository.Tx(...)。
// service 不持有 DB 句柄——事务连接全程由 repository 管理。
// UploadDir / JWTSecret / JWTIssuer / rdb 均为只读依赖，由 main 在启动时注入。
// 说明：rdb（Redis）用于验证码、登录限流、令牌吊销与认证画像缓存，属会话/缓存基础设施。
type Service struct {
	UploadDir        string
	JWTSecret        string
	JWTIssuer        string
	rdb              *redis.Client
	AuthRepo         *repository.AuthRepo
	UserRepo         *repository.UserRepo
	CustomerRepo     *repository.CustomerRepo
	AssetRepo        *repository.AssetRepo
	CalendarRepo     *repository.CalendarRepo
	NotificationRepo *repository.NotificationRepo
	SettingsRepo     *repository.SettingsRepo
	PackageRepo      *repository.PackageRepo
	LeadRepo         *repository.LeadRepo
	OrderRepo        *repository.OrderRepo
	DeliveryRepo     *repository.DeliveryRepo
	FinanceRepo      *repository.FinanceRepo
	DashboardRepo    *repository.DashboardRepo
	UploadRepo       *repository.UploadRepo
	// 三端（小程序/APP/H5）新增实体
	RescheduleRepo    *repository.OrderRescheduleRepo
	ReviewRepo        *repository.ReviewRepo
	CustomRequestRepo *repository.CustomRequestRepo
	AddonRepo         *repository.AddonRepo
	LeadExtraRepo     *repository.LeadExtraRepo
	SlotTemplateRepo  *repository.SlotTemplateRepo
	StudioSettingRepo *repository.StudioSettingRepo
	DeviceRepo        *repository.DeviceRepo
}

// New 构造业务服务。依赖（上传目录 / JWT 参数 / Redis 客户端）由组合根注入，
// service 不自行获取任何全局基础设施句柄（#40）。
func New(uploadDir, jwtSecret, jwtIssuer string, rdb *redis.Client) *Service {
	return &Service{
		UploadDir:         uploadDir,
		JWTSecret:         jwtSecret,
		JWTIssuer:         jwtIssuer,
		rdb:               rdb,
		AuthRepo:          repository.NewAuthRepo(),
		UserRepo:          repository.NewUserRepo(),
		CustomerRepo:      repository.NewCustomerRepo(),
		AssetRepo:         repository.NewAssetRepo(),
		CalendarRepo:      repository.NewCalendarRepo(),
		NotificationRepo:  repository.NewNotificationRepo(),
		SettingsRepo:      repository.NewSettingsRepo(),
		PackageRepo:       repository.NewPackageRepo(),
		LeadRepo:          repository.NewLeadRepo(),
		OrderRepo:         repository.NewOrderRepo(),
		DeliveryRepo:      repository.NewDeliveryRepo(),
		FinanceRepo:       repository.NewFinanceRepo(),
		DashboardRepo:     repository.NewDashboardRepo(),
		UploadRepo:        repository.NewUploadRepo(),
		RescheduleRepo:    repository.NewOrderRescheduleRepo(),
		ReviewRepo:        repository.NewReviewRepo(),
		CustomRequestRepo: repository.NewCustomRequestRepo(),
		AddonRepo:         repository.NewAddonRepo(),
		LeadExtraRepo:     repository.NewLeadExtraRepo(),
		SlotTemplateRepo:  repository.NewSlotTemplateRepo(),
		StudioSettingRepo: repository.NewStudioSettingRepo(),
		DeviceRepo:        repository.NewDeviceRepo(),
	}
}

// tenant 按 company_id 过滤的查询会话已下沉到 repository 的 Repo.tenant，service 不再持有 DB 句柄。
// 通用纯函数（金额取整/业务编码/退款比例/订单状态机）已下沉到 internal/domain，service 不再重复实现。

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

// orDefaultInt int 版零值回退（创建取默认值、更新保持库中原值）。
// 注意：当 0 是合法业务取值时不可使用本函数，应改用指针参数区分"未传"。
func orDefaultInt(v, def int) int {
	if v == 0 {
		return def
	}
	return v
}

// orDefaultFloat float64 版零值回退（更新场景保持库中原值）
func orDefaultFloat(v, def float64) float64 {
	if v == 0 {
		return def
	}
	return v
}

// orDefaultEnum 枚举零值回退：v==0 时取 def。
// 创建场景 def 传默认状态，更新场景 def 传库中当前值（实现"未传则保持原值"）。
func orDefaultEnum[T ~int](v, def T) T {
	if v == 0 {
		return def
	}
	return v
}

// requirePerm 校验操作者是否具备指定权限点。
//
// 用途：**路由级无法区分的字段级动作**——同一个接口既能做普通编辑、又能触发高敏动作时，
// 由 service 层按字段差异追加校验。目前用于 lead/update 变更归属人（= 分配线索，需 lead:assign）
// 与 lead/update 变更预算（需 lead:update，由路由级已覆盖）。
//
// 注意与路由级 mw.Perm 的分工：能靠路由区分的动作一律用路由级挂载（就近可见、无需读业务代码），
// 只有当"同一接口内的不同字段对应不同权限点"时才用本函数。admin 角色短路放行，
// 判定口径与 middleware.hasPerm 保持一致。
func requirePerm(op Operator, want domain.Perm) error {
	if op.RoleCode == domain.RoleCodeAdmin {
		return nil
	}
	if !domain.HasPerm(op.Permissions, want) {
		return errs.Forbidden(errs.ErrForbidden)
	}
	return nil
}

// orderLog 构造订单操作日志实体。
// 必须带上 TenantBase：OrderLog 内嵌 TenantBase，写入走 conn().Create（不注入租户），
// 而读取走 tenant(companyID) 过滤。漏填 CompanyID 会导致日志以 company_id=0 入库，
// 按真实租户查询时永远为空——订单流水追溯整体失效。
func orderLog(orderID int64, action string, from, to interface{}, content string, op Operator) *model.OrderLog {
	return &model.OrderLog{
		TenantBase: model.TenantBase{
			Base:      model.Base{CreatedBy: op.UserID, UpdatedBy: op.UserID},
			CompanyID: op.CompanyID,
		},
		OrderID:      orderID,
		Action:       action,
		FromStatus:   fmt.Sprintf("%v", from),
		ToStatus:     fmt.Sprintf("%v", to),
		Content:      content,
		OperatorID:   op.UserID,
		OperatorName: op.Username,
	}
}

func (s *Service) writeOrderLog(ctx context.Context, orderID int64, action string, from, to interface{}, content string, op Operator) error {
	return s.OrderRepo.CreateLog(ctx, orderLog(orderID, action, from, to, content, op))
}

func (s *Service) writeOrderLogTx(ctx context.Context, tx *gorm.DB, orderID int64, action string, from, to interface{}, content string, op Operator) error {
	return s.OrderRepo.WithTx(tx).CreateLog(ctx, orderLog(orderID, action, from, to, content, op))
}
