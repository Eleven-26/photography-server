package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"

	"photography-server/internal/domain"
	"photography-server/internal/enum"
	"photography-server/internal/model"
	"photography-server/internal/pkg/errs"
	"photography-server/internal/presentation/dto"
)

// clientCatalog 客户端（H5/小程序）公开目录能力：
// 套餐浏览、工作室信息、可约时段、定制需求提交。companyID 由路由/租户上下文提供，
// 无需客户登录即可访问（预约前浏览）。

// ResolveCompanyBySlug 按预约主页短链标识（slug）反查公司 ID——客户端公开接口的租户定位，
// 替代"客户端直传 company_id"（裸数字可被遍历枚举全平台工作室，审查报告 #29）。
// 返回 companyID=0 表示 slug 不存在或未配置；其他错误原样返回。
func (s *Service) ResolveCompanyBySlug(ctx context.Context, slug string) (int64, error) {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return 0, nil
	}
	st, err := s.StudioSettingRepo.GetBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, nil
		}
		return 0, err
	}
	return st.CompanyID, nil
}

// ClientPackages 已上架套餐列表（客户预约主页）
func (s *Service) ClientPackages(ctx context.Context, companyID int64, page, pageSize int, category string) ([]model.Package, int64, error) {
	// 仅返回已上架（PackageStatusActive=2）
	return s.PackageRepo.List(ctx, companyID, page, pageSize, "", strconv.Itoa(int(enum.PackageStatusActive)), category)
}

// ClientPackageDetail 套餐详情（含适用人群/交付说明等客户端展示字段）
func (s *Service) ClientPackageDetail(ctx context.Context, companyID, id int64) (*model.Package, error) {
	pkg, err := s.PackageRepo.GetByID(ctx, companyID, id)
	if err != nil {
		return nil, errs.NotFound(errs.ErrPackageNotFound)
	}
	if pkg.Status != enum.PackageStatusActive {
		return nil, errs.BadRequest("套餐已下架")
	}
	return pkg, nil
}

// ClientStudioInfo 工作室预约主页信息（不存在自动创建默认配置）
func (s *Service) ClientStudioInfo(ctx context.Context, companyID int64) (*model.StudioSetting, error) {
	return s.StudioSettingRepo.GetOrCreate(ctx, companyID)
}

// ClientSlots 指定日期的可约时段：按档期模板展开，并排除已被档期锁占用的时段。
// 返回时段列表 + 是否可约。
func (s *Service) ClientSlots(ctx context.Context, companyID int64, date string, photographerID int64) ([]dto.ClientSlot, error) {
	if date == "" {
		return nil, errs.BadRequest("请选择日期")
	}
	d, err := domain.ParseShootDate(date) // #16：统一本地时区解析
	if err != nil {
		return nil, errs.BadRequest("日期格式错误，应为 2006-01-02")
	}
	// 过去日期不可约
	if d.Before(time.Now().Truncate(24 * time.Hour)) {
		return nil, errs.BadRequest("不可选择过去的日期")
	}

	templates, err := s.SlotTemplateRepo.List(ctx, companyID, photographerID)
	if err != nil {
		return nil, err
	}
	weekday := int(d.Weekday())

	blocks, err := s.CalendarRepo.List(ctx, companyID, date, date, 0)
	if err != nil {
		return nil, err
	}
	occupied := map[string]bool{}
	for _, b := range blocks {
		if b.Status == enum.BlockStatusCancelled {
			continue
		}
		occupied[b.TimeRange] = true
	}

	slots := make([]dto.ClientSlot, 0, len(templates))
	for _, t := range templates {
		if t.Weekday != weekday {
			continue
		}
		tr := fmt.Sprintf("%s-%s", t.StartTime, t.EndTime)
		slots = append(slots, dto.ClientSlot{
			StartTime: t.StartTime,
			EndTime:   t.EndTime,
			Available: !occupied[tr] && !occupied[t.StartTime],
		})
	}
	return slots, nil
}

// ClientSubmitCustomRequest 提交定制需求（登录或游客均可提交）
func (s *Service) ClientSubmitCustomRequest(ctx context.Context, companyID int64, cu *ClientUser, req dto.ClientCustomRequestReq) (*model.CustomRequest, error) {
	if req.ProjectType == "" {
		return nil, errs.BadRequest("请选择拍摄类型")
	}
	if cu == nil && req.Mobile == "" {
		return nil, errs.BadRequest("请填写联系电话")
	}
	m := model.CustomRequest{
		TenantBase:   model.TenantBase{Base: model.Base{CreatedAt: time.Now(), UpdatedAt: time.Now()}, CompanyID: companyID},
		ProjectType:  req.ProjectType,
		ExpectedDate: req.ExpectedDate,
		Location:     req.Location,
		BudgetMin:    req.BudgetMin,
		BudgetMax:    req.BudgetMax,
		Detail:       req.Detail,
		Images:       req.Images,
		Status:       enum.CustomRequestPending,
	}
	if cu != nil {
		m.CustomerID = cu.CustomerID
		m.Name = orDefault(req.Name, cu.Name)
		m.Mobile = orDefault(req.Mobile, cu.Mobile)
	} else {
		m.Name = req.Name
		m.Mobile = req.Mobile
	}
	if err := s.CustomRequestRepo.Create(ctx, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

// ClientCustomRequests 我的定制需求列表（登录客户）
func (s *Service) ClientCustomRequests(ctx context.Context, cu *ClientUser, page, pageSize int) ([]model.CustomRequest, int64, error) {
	return s.CustomRequestRepo.List(ctx, cu.CompanyID, page, pageSize, 0, cu.CustomerID)
}

// clientReschedulePolicy 从工作室设置读取改期政策（缺省用 domain 内置默认值）
func (s *Service) clientReschedulePolicy(ctx context.Context, companyID int64) domain.ReschedulePolicy {
	p := domain.ReschedulePolicy{}
	if st, err := s.StudioSettingRepo.GetByCompany(ctx, companyID); err == nil {
		p.FreeHours = st.RescheduleFreeHours
		p.MinHours = st.RescheduleMinHours
		p.FeeRate = st.RescheduleFeeRate
	}
	return p
}
