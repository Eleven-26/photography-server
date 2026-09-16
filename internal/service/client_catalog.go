package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"

	"photography-server/internal/contract"
	"photography-server/internal/domain"
	"photography-server/internal/enum"
	"photography-server/internal/model"
	"photography-server/internal/pkg/errs"
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
		return nil, errs.BadRequest(errs.ErrPackageOffline)
	}
	return pkg, nil
}

// ClientStudioInfo 工作室预约主页信息（不存在自动创建默认配置）
func (s *Service) ClientStudioInfo(ctx context.Context, companyID int64) (*model.StudioSetting, error) {
	return s.StudioSettingRepo.GetOrCreate(ctx, companyID)
}

// ClientSlots 指定日期的可约时段：按档期模板展开，并排除已被档期锁占用的时段。
// 返回时段列表 + 是否可约。
func (s *Service) ClientSlots(ctx context.Context, companyID int64, date string, photographerID int64) ([]contract.ClientSlot, error) {
	if date == "" {
		return nil, errs.BadRequest(errs.ErrDateRequired)
	}
	d, err := domain.ParseShootDate(date) // #16：统一本地时区解析
	if err != nil {
		return nil, errs.BadRequest(errs.ErrDateFormatInvalid)
	}
	// 过去日期不可约
	if d.Before(time.Now().Truncate(24 * time.Hour)) {
		return nil, errs.BadRequest(errs.ErrDateInPast)
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

	slots := make([]contract.ClientSlot, 0, len(templates))
	for _, t := range templates {
		if t.Weekday != weekday {
			continue
		}
		tr := fmt.Sprintf("%s-%s", t.StartTime, t.EndTime)
		slots = append(slots, contract.ClientSlot{
			StartTime: t.StartTime,
			EndTime:   t.EndTime,
			Available: !occupied[tr] && !occupied[t.StartTime],
		})
	}
	return slots, nil
}

// ClientSubmitCustomRequest 提交定制需求（登录或游客均可提交）。
//
// 摄影师归属（2026-09-15 补齐，此前该链路是断的）：
//   - req.PhotographerID = 客户在 H5 定制需求页「选择门店 → 选择摄影师」里的**显式选择**；
//   - staffID = 分享链接带入的分享人（?staff_id= / X-Staff-Id 头，见 h5.go → staffFrom）。
//
// 显式选择优先，分享链接兜底 —— 客户从个人中心进来（URL 无 staff_id）也能指定摄影师，
// 而不是只能靠链接归属。两者都缺失时落 store_id 对应门店或公共池（store_id=0），
// 由工作室后续指派/认领。
func (s *Service) ClientSubmitCustomRequest(ctx context.Context, companyID int64, cu *ClientUser, req contract.ClientCustomRequestReq, staffID int64) (*model.CustomRequest, error) {
	if req.ProjectType == "" {
		return nil, errs.BadRequest(errs.ErrBookingProjectTypeRequired)
	}
	if cu == nil && req.Mobile == "" {
		return nil, errs.BadRequest(errs.ErrBookingMobileRequired)
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
	// 门店归属（独立接单）：提交时指定目标门店则直接落店；非法值报错，
	// 不静默改写为 0——静默降级会让"客户以为提交给了某店、门店却看不到"。
	if req.StoreID > 0 {
		stores, err := s.UserRepo.ListStores(ctx, companyID)
		if err != nil {
			return nil, err
		}
		valid := false
		for _, st := range stores {
			if st.ID == req.StoreID {
				valid = true
				break
			}
		}
		if !valid {
			return nil, errs.BadRequest(errs.ErrBookingStoreInvalid)
		}
		m.StoreID = req.StoreID
	}
	// 摄影师归属：显式选择优先，分享链接带入兜底。
	// 校验「存在 + 属于当前机构 + 启用中」：已停用的员工不能再被指定（接不了单）；
	// 跨租户 ID 会被 tenant 过滤掉而查不到，与「不存在」归为同一文案，不泄露他租户信息。
	pid := req.PhotographerID
	if pid <= 0 {
		pid = staffID
	}
	if pid > 0 {
		u, err := s.UserRepo.GetByID(ctx, companyID, pid)
		if err != nil || u == nil || u.Status != 1 {
			return nil, errs.BadRequest(errs.ErrBookingStaffNotAvailable)
		}
		// 门店与摄影师必须自洽：前端两级联动保证一致，后端不信任客户端 ——
		// 两者都给且不同，说明请求被改造过，直接拒绝而不是静默取其一（同门店归属的既定口径：
		// 不静默降级，否则会出现"客户以为指定了 A 店摄影师、A 店却看不到"）。
		if m.StoreID > 0 && u.StoreID > 0 && m.StoreID != u.StoreID {
			return nil, errs.BadRequest(errs.ErrBookingStaffNotInStore)
		}
		m.PhotographerID = u.ID
		m.Photographer = u.Nickname
		if m.StoreID == 0 {
			m.StoreID = u.StoreID // 只选了摄影师：按该摄影师所属门店落店
		}
	}
	if cu != nil {
		m.CustomerID = cu.CustomerID
		m.Name = orDefault(req.Name, cu.Name)
		m.Mobile = orDefault(req.Mobile, cu.Mobile)
	} else {
		m.Name = req.Name
		m.Mobile = req.Mobile
	}
	// 客户主体对齐：游客提交（或已登录但未绑定客户档案）时，按手机号在客户表查档，
	// 查不到则自动建档 —— 否则需求只带姓名手机号，后续转订单/交付/通知挂不到客户身上。
	if m.CustomerID == 0 {
		c, err := s.FindOrCreateCustomerByMobile(ctx, companyID, m.Name, m.Mobile, "定制需求")
		if err != nil {
			return nil, err
		}
		if c != nil {
			m.CustomerID = c.ID
		}
	}
	if err := s.CustomRequestRepo.Create(ctx, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

// ClientCustomRequests 我的定制需求列表（登录客户）
func (s *Service) ClientCustomRequests(ctx context.Context, cu *ClientUser, page, pageSize int) ([]model.CustomRequest, int64, error) {
	// 末位 photographerID=0：客户侧不按摄影师过滤（那是管理端筛选维度）
	return s.CustomRequestRepo.List(ctx, cu.CompanyID, page, pageSize, 0, cu.CustomerID, 0)
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
