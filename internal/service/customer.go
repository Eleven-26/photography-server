package service

import (
	"context"

	"photography-server/internal/domain"
	"photography-server/internal/enum"
	"photography-server/internal/model"
	"photography-server/internal/pkg/errs"
	"photography-server/internal/pkg/logger"
	"photography-server/internal/presentation/dto"
)

func (s *Service) ListCustomers(ctx context.Context, op Operator, page, pageSize int, keyword string) ([]model.Customer, int64, error) {
	return s.CustomerRepo.List(ctx, op.CompanyID, page, pageSize, keyword)
}

func (s *Service) GetCustomer(ctx context.Context, op Operator, id int64) (*model.Customer, error) {
	c, err := s.CustomerRepo.GetByID(ctx, op.CompanyID, id)
	if err != nil {
		return nil, errs.NotFound(errs.ErrCustomerNotFound)
	}
	// 满意度为派生字段（该客户订单评价均分）。取数失败不影响主档案，
	// 降级为 0 由前端显示「暂无评价」，不伪造分数。
	if rating, err := s.CustomerRepo.AvgRating(ctx, op.CompanyID, id); err != nil {
		logger.Warnf("GetCustomer: AvgRating failed, customerID=%d, err=%v", id, err)
	} else {
		c.Satisfaction = rating
	}
	return c, nil
}

func (s *Service) CreateCustomer(ctx context.Context, op Operator, req dto.CustomerCreateReq) (*model.Customer, error) {
	c := model.Customer{
		TenantBase: model.TenantBase{
			Base:      model.Base{CreatedBy: op.UserID, UpdatedBy: op.UserID},
			CompanyID: op.CompanyID,
		},
		Code:               domain.GenCode("CU"),
		StoreID:            orDefaultInt64(req.StoreID, op.StoreID),
		Name:               req.Name,
		Mobile:             req.Mobile,
		Wechat:             req.Wechat,
		Gender:             req.Gender,
		Birthday:           strPtr(req.Birthday),
		Level:              orDefaultEnum(req.Level, enum.CustomerLevelNormal),
		Source:             req.Source,
		Tags:               req.Tags,
		Status:             orDefaultEnum(req.Status, enum.CustomerStatusPotential),
		Remark:             req.Remark,
		Avatar:             req.Avatar,
		AllowNotifications: allowNotifyPtr(req.AllowNotifications),
		PreferStyle:        req.PreferStyle,
		PreferScene:        req.PreferScene,
	}
	if err := s.CustomerRepo.Create(ctx, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *Service) UpdateCustomer(ctx context.Context, op Operator, id int64, req dto.CustomerUpdateReq) error {
	cur, err := s.CustomerRepo.GetByID(ctx, op.CompanyID, id)
	if err != nil {
		return errs.NotFound(errs.ErrCustomerNotFound)
	}
	updates := map[string]interface{}{
		"store_id":     req.StoreID,
		"name":         req.Name,
		"mobile":       req.Mobile,
		"wechat":       req.Wechat,
		"gender":       req.Gender,
		"birthday":     req.Birthday,
		"level":        orDefaultEnum(req.Level, cur.Level),
		"source":       req.Source,
		"tags":         req.Tags,
		"status":       orDefaultEnum(req.Status, cur.Status),
		"remark":       req.Remark,
		"avatar":       req.Avatar,
		"prefer_style": req.PreferStyle,
		"prefer_scene": req.PreferScene,
		"updated_by":   op.UserID,
	}
	// 通知许可是开关：nil=未传（保持原值），显式 0 必须能落库，
	// 因此不能走 orDefault（否则「关闭通知」会被静默回退成原值）。
	if req.AllowNotifications != nil {
		updates["allow_notifications"] = *req.AllowNotifications
	}
	if err := s.CustomerRepo.Update(ctx, op.CompanyID, id, updates); err != nil {
		return err
	}
	// 状态（含停用/流失）变更：失效客户认证画像缓存，使停用即时生效（#30）
	invalidateCustomerCache(ctx, op.CompanyID, id)
	return nil
}

func (s *Service) DeleteCustomer(ctx context.Context, op Operator, id int64) error {
	if err := s.CustomerRepo.Delete(ctx, op.CompanyID, id); err != nil {
		return err
	}
	invalidateCustomerCache(ctx, op.CompanyID, id)
	return nil
}

// allowNotifyPtr 通知许可：创建时未传视为允许(1)；显式传值时原样保留。
func allowNotifyPtr(v *int) *int {
	if v == nil {
		one := 1
		return &one
	}
	return v
}

func (s *Service) GetCustomerStats(ctx context.Context, op Operator) (*dto.CustomerStatsResp, error) {
	st, err := s.CustomerRepo.GetStats(ctx, op.CompanyID)
	if err != nil {
		return nil, err
	}
	// repository 返回自持结构，service 负责映射为对外 dto
	return &dto.CustomerStatsResp{
		Total:           st.Total,
		Potential:       st.Potential,
		Active:          st.Active,
		Inactive:        st.Inactive,
		GoldUp:          st.GoldUp,
		NewThisMonth:    st.NewThisMonth,
		RepurchaseCount: st.RepurchaseCount,
		RepurchaseRate:  st.RepurchaseRate,
	}, nil
}
