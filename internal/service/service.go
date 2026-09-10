package service

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"photography-server/internal/domain"
	"photography-server/internal/model"
	"photography-server/internal/repository"
)

// Operator 当前操作人（由认证中间件注入）
// 重构 #32：类型别名，实际定义在 domain 包
type Operator = domain.Operator

// Service 业务服务根结构，按领域拆分到不同文件。
// 分层纪律：service 只依赖 repository（数据访问唯一入口），
// 不持有任何基础设施句柄；开启事务统一走 repository.Tx(...)。
// JWTSecret/JWTIssuer：客户端三端签发 JWT 所需（由 config.JWT 注入，只读）。
type Service struct {
	UploadDir        string
	JWTSecret        string
	JWTIssuer        string
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

func New(uploadDir, jwtSecret, jwtIssuer string) *Service {
	return &Service{
		UploadDir:         uploadDir,
		JWTSecret:         jwtSecret,
		JWTIssuer:         jwtIssuer,
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
