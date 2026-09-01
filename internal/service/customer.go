package service

import (
	"photography-server/internal/common"
	"photography-server/internal/enum"
	"photography-server/internal/model"
	"photography-server/internal/pkg/errs"
	"photography-server/internal/presentation/dto"
)

func (s *Service) ListCustomers(op Operator, page, pageSize int, keyword string) ([]model.Customer, int64, error) {
	return s.CustomerRepo.List(op.CompanyID, page, pageSize, keyword)
}

func (s *Service) GetCustomer(op Operator, id int64) (*model.Customer, error) {
	c, err := s.CustomerRepo.GetByID(op.CompanyID, id)
	if err != nil {
		return nil, errs.NotFound(common.ErrCustomerNotFound)
	}
	return c, nil
}

func (s *Service) CreateCustomer(op Operator, req dto.CustomerCreateReq) (*model.Customer, error) {
	c := model.Customer{
		TenantBase: model.TenantBase{
			Base:      model.Base{CreatedBy: op.UserID, UpdatedBy: op.UserID},
			CompanyID: op.CompanyID,
		},
		Code:     genCode("CU"),
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
	if err := s.DB().Create(&c).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *Service) UpdateCustomer(op Operator, id int64, req dto.CustomerUpdateReq) error {
	_, err := s.CustomerRepo.GetByID(op.CompanyID, id)
	if err != nil {
		return errs.NotFound(common.ErrCustomerNotFound)
	}
	return s.CustomerRepo.Update(op.CompanyID, id, map[string]interface{}{
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

func (s *Service) DeleteCustomer(op Operator, id int64) error {
	return s.CustomerRepo.Delete(op.CompanyID, id)
}

func (s *Service) GetCustomerStats(op Operator) (*dto.CustomerStatsResp, error) {
	return s.CustomerRepo.GetStats(op.CompanyID)
}
