package service

import (
	"fmt"

	"gorm.io/gorm"

	"photography-server/internal/model"
	"photography-server/internal/repository"
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

// Service 业务服务根结构，按领域拆分到不同文件。
// 分层纪律：service 只依赖 repository（数据访问唯一入口），
// 不持有任何基础设施句柄；开启事务统一走 repository.Tx(...)。
type Service struct {
	UploadDir        string
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
}

func New(uploadDir string) *Service {
	return &Service{
		UploadDir:        uploadDir,
		AuthRepo:         repository.NewAuthRepo(),
		UserRepo:         repository.NewUserRepo(),
		CustomerRepo:     repository.NewCustomerRepo(),
		AssetRepo:        repository.NewAssetRepo(),
		CalendarRepo:     repository.NewCalendarRepo(),
		NotificationRepo: repository.NewNotificationRepo(),
		SettingsRepo:     repository.NewSettingsRepo(),
		PackageRepo:      repository.NewPackageRepo(),
		LeadRepo:         repository.NewLeadRepo(),
		OrderRepo:        repository.NewOrderRepo(),
		DeliveryRepo:     repository.NewDeliveryRepo(),
		FinanceRepo:      repository.NewFinanceRepo(),
		DashboardRepo:    repository.NewDashboardRepo(),
		UploadRepo:       repository.NewUploadRepo(),
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

func (s *Service) writeOrderLog(orderID int64, action string, from, to interface{}, content string, op Operator) error {
	log := model.OrderLog{
		OrderID:      orderID,
		Action:       action,
		FromStatus:   fmt.Sprintf("%v", from),
		ToStatus:     fmt.Sprintf("%v", to),
		Content:      content,
		OperatorID:   op.UserID,
		OperatorName: op.Username,
	}
	return s.OrderRepo.CreateLog(&log)
}

func (s *Service) writeOrderLogTx(tx *gorm.DB, orderID int64, action string, from, to interface{}, content string, op Operator) error {
	log := model.OrderLog{
		OrderID:      orderID,
		Action:       action,
		FromStatus:   fmt.Sprintf("%v", from),
		ToStatus:     fmt.Sprintf("%v", to),
		Content:      content,
		OperatorID:   op.UserID,
		OperatorName: op.Username,
	}
	return s.OrderRepo.WithTx(tx).CreateLog(&log)
}
