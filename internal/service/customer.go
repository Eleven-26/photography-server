package service

import (
	"errors"

	"gorm.io/gorm"

	"photography-server/internal/common"
	"photography-server/internal/model"
	"photography-server/internal/pkg/errs"
	"photography-server/internal/presentation/dto"
)

func (s *Service) ListCustomers(op Operator, page, pageSize int, keyword, status, level string) ([]model.Customer, int64, error) {
	return s.CustomerRepo.List(op.CompanyID, page, pageSize, keyword, status, level)
}

func (s *Service) GetCustomer(op Operator, id int64) (*model.Customer, error) {
	c, err := s.CustomerRepo.GetByID(op.CompanyID, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.NotFound(common.ErrCustomerNotFound)
		}
		return nil, err
	}
	return c, nil
}

// CreateCustomerTx 在指定事务内创建客户（供线索转客户等场景复用）
func (s *Service) CreateCustomerTx(tx *gorm.DB, op Operator, req dto.CustomerCreateReq) (*model.Customer, error) {
	c := model.Customer{
		TenantBase: model.TenantBase{
			Base:      model.Base{CreatedBy: op.UserID, UpdatedBy: op.UserID},
			CompanyID: op.CompanyID,
		},
		Code:     genCode("CU"),
		StoreID:  req.StoreID,
		Name:     req.Name,
		Mobile:   req.Mobile,
		Wechat:   req.Wechat,
		Gender:   req.Gender,
		Birthday: strPtr(req.Birthday),
		Level:    orDefault(req.Level, model.CustomerLevelNormal),
		Source:   req.Source,
		Tags:     req.Tags,
		Status:   orDefault(req.Status, model.CustomerStatusPotential),
		Remark:   req.Remark,
		Avatar:   req.Avatar,
	}
	if err := s.CustomerRepo.Create(tx, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *Service) CreateCustomer(op Operator, req dto.CustomerCreateReq) (*model.Customer, error) {
	return s.CreateCustomerTx(s.DB(), op, req)
}

func (s *Service) UpdateCustomer(op Operator, id int64, req dto.CustomerUpdateReq) error {
	_, err := s.CustomerRepo.GetByID(op.CompanyID, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errs.NotFound(common.ErrCustomerNotFound)
		}
		return err
	}
	return s.CustomerRepo.Update(op.CompanyID, id, map[string]interface{}{
		"store_id": req.StoreID, "name": req.Name, "mobile": req.Mobile,
		"wechat": req.Wechat, "gender": req.Gender, "birthday": req.Birthday,
		"level": req.Level, "source": req.Source, "tags": req.Tags,
		"status": req.Status, "remark": req.Remark, "avatar": req.Avatar,
		"updated_by": op.UserID,
	})
}

func (s *Service) DeleteCustomer(op Operator, id int64) error {
	return s.CustomerRepo.Delete(op.CompanyID, id)
}

func (s *Service) CustomerStats(op Operator) (*dto.CustomerStatsResp, error) {
	total, active, goldUp, err := s.CustomerRepo.Stats(op.CompanyID)
	if err != nil {
		return nil, err
	}
	return &dto.CustomerStatsResp{
		Total:  total,
		Active: active,
		GoldUp: goldUp,
	}, nil
}
