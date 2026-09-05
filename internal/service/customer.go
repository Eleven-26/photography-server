package service

import (
	"context"

	"photography-server/internal/domain"
	"photography-server/internal/enum"
	"photography-server/internal/model"
	"photography-server/internal/pkg/errs"
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
	return c, nil
}

func (s *Service) CreateCustomer(ctx context.Context, op Operator, req dto.CustomerCreateReq) (*model.Customer, error) {
	c := model.Customer{
		TenantBase: model.TenantBase{
			Base:      model.Base{CreatedBy: op.UserID, UpdatedBy: op.UserID},
			CompanyID: op.CompanyID,
		},
		Code:     domain.GenCode("CU"),
		StoreID:  orDefaultInt64(req.StoreID, op.StoreID),
		Name:     req.Name,
		Mobile:   req.Mobile,
		Wechat:   req.Wechat,
		Gender:   req.Gender,
		Birthday: strPtr(req.Birthday),
		Level:    enum.CustomerLevelNormal,
		Source:   req.Source,
		Tags:     req.Tags,
		Status:   enum.CustomerStatusPotential,
		Remark:   req.Remark,
		Avatar:   req.Avatar,
	}
	if err := s.CustomerRepo.Create(ctx, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *Service) UpdateCustomer(ctx context.Context, op Operator, id int64, req dto.CustomerUpdateReq) error {
	_, err := s.CustomerRepo.GetByID(ctx, op.CompanyID, id)
	if err != nil {
		return errs.NotFound(errs.ErrCustomerNotFound)
	}
	return s.CustomerRepo.Update(ctx, op.CompanyID, id, map[string]interface{}{
		"store_id":   req.StoreID,
		"name":       req.Name,
		"mobile":     req.Mobile,
		"wechat":     req.Wechat,
		"gender":     req.Gender,
		"birthday":   req.Birthday,
		"level":      req.Level,
		"source":     req.Source,
		"tags":       req.Tags,
		"status":     req.Status,
		"remark":     req.Remark,
		"avatar":     req.Avatar,
		"updated_by": op.UserID,
	})
}

func (s *Service) DeleteCustomer(ctx context.Context, op Operator, id int64) error {
	return s.CustomerRepo.Delete(ctx, op.CompanyID, id)
}

func (s *Service) GetCustomerStats(ctx context.Context, op Operator) (*dto.CustomerStatsResp, error) {
	st, err := s.CustomerRepo.GetStats(ctx, op.CompanyID)
	if err != nil {
		return nil, err
	}
	// repository 返回自持结构，service 负责映射为对外 dto
	return &dto.CustomerStatsResp{
		Total:  st.Total,
		Active: st.Active,
		GoldUp: st.GoldUp,
	}, nil
}
