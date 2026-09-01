package service

import (
	"errors"

	"gorm.io/gorm"

	"photography-server/internal/common"
	"photography-server/internal/model"
	"photography-server/internal/pkg/errs"
	"photography-server/internal/presentation/dto"
)

type Workspace struct {
	Company  model.SysCompany      `json:"company"`
	Stores   []model.SysStore      `json:"stores"`
	Roles    []model.SysRole       `json:"roles"`
	Payments []model.PaymentMethod `json:"payment_methods"`
}

func (s *Service) Workspace(op Operator) (*Workspace, error) {
	c, err := s.SettingsRepo.GetCompany(op.CompanyID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.NotFound(common.ErrCompanyNotFound)
		}
		return nil, err
	}
	w := &Workspace{Company: *c}
	w.Stores, err = s.SettingsRepo.ListStores(op.CompanyID)
	if err != nil {
		return nil, err
	}
	w.Roles, err = s.SettingsRepo.ListRoles(op.CompanyID)
	if err != nil {
		return nil, err
	}
	w.Payments, err = s.SettingsRepo.ListPaymentMethods(op.CompanyID)
	if err != nil {
		return nil, err
	}
	return w, nil
}

func (s *Service) UpdateCompany(op Operator, req dto.CompanyUpdateReq) error {
	return s.SettingsRepo.UpdateCompany(op.CompanyID, map[string]interface{}{
		"name": req.Name, "logo": req.Logo,
		"contact_name": req.ContactName, "contact_phone": req.ContactPhone,
		"address": req.Address, "updated_by": op.UserID,
	})
}

// -------- 收款方式 --------

func (s *Service) ListPaymentMethods(op Operator) ([]model.PaymentMethod, error) {
	return s.SettingsRepo.ListPaymentMethods(op.CompanyID)
}

func (s *Service) CreatePaymentMethod(op Operator, req dto.PaymentMethodReq) error {
	m := model.PaymentMethod{
		TenantBase: model.TenantBase{
			Base:      model.Base{CreatedBy: op.UserID, UpdatedBy: op.UserID},
			CompanyID: op.CompanyID,
		},
		Name: req.Name, Type: req.Type, AccountName: req.AccountName,
		AccountNo: req.AccountNo, Qrcode: req.Qrcode, Status: req.Status, Sort: req.Sort,
	}
	return s.SettingsRepo.CreatePaymentMethod(&m)
}

func (s *Service) UpdatePaymentMethod(op Operator, id int64, req dto.PaymentMethodReq) error {
	_, err := s.SettingsRepo.GetPaymentMethodByID(op.CompanyID, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errs.NotFound(common.ErrPaymentMethodNotFound)
		}
		return err
	}
	return s.SettingsRepo.UpdatePaymentMethod(op.CompanyID, id, map[string]interface{}{
		"name": req.Name, "type": req.Type, "account_name": req.AccountName,
		"account_no": req.AccountNo, "qrcode": req.Qrcode, "status": req.Status,
		"sort": req.Sort, "updated_by": op.UserID,
	})
}

func (s *Service) DeletePaymentMethod(op Operator, id int64) error {
	return s.SettingsRepo.DeletePaymentMethod(op.CompanyID, id)
}

// -------- 操作日志 --------

func (s *Service) ListOperationLogs(op Operator, page, pageSize int) ([]model.SysOperationLog, int64, error) {
	return s.SettingsRepo.ListOperationLogs(op.CompanyID, page, pageSize)
}
