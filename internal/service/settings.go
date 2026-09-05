package service

import (
	"context"
	"errors"

	"gorm.io/gorm"

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

func (s *Service) Workspace(ctx context.Context, op Operator) (*Workspace, error) {
	c, err := s.SettingsRepo.GetCompany(ctx, op.CompanyID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.NotFound(errs.ErrCompanyNotFound)
		}
		return nil, err
	}
	w := &Workspace{Company: *c}
	w.Stores, err = s.SettingsRepo.ListStores(ctx, op.CompanyID)
	if err != nil {
		return nil, err
	}
	w.Roles, err = s.SettingsRepo.ListRoles(ctx, op.CompanyID)
	if err != nil {
		return nil, err
	}
	w.Payments, err = s.SettingsRepo.ListPaymentMethods(ctx, op.CompanyID)
	if err != nil {
		return nil, err
	}
	return w, nil
}

func (s *Service) UpdateCompany(ctx context.Context, op Operator, req dto.CompanyUpdateReq) error {
	return s.SettingsRepo.UpdateCompany(ctx, op.CompanyID, map[string]interface{}{
		"name": req.Name, "logo": req.Logo,
		"contact_name": req.ContactName, "contact_phone": req.ContactPhone,
		"address": req.Address, "updated_by": op.UserID,
	})
}

// -------- 收款方式 --------

func (s *Service) ListPaymentMethods(ctx context.Context, op Operator) ([]model.PaymentMethod, error) {
	return s.SettingsRepo.ListPaymentMethods(ctx, op.CompanyID)
}

func (s *Service) CreatePaymentMethod(ctx context.Context, op Operator, req dto.PaymentMethodReq) error {
	m := model.PaymentMethod{
		TenantBase: model.TenantBase{
			Base:      model.Base{CreatedBy: op.UserID, UpdatedBy: op.UserID},
			CompanyID: op.CompanyID,
		},
		Name: req.Name, Type: req.Type, AccountName: req.AccountName,
		AccountNo: req.AccountNo, Qrcode: req.Qrcode, Status: req.Status, Sort: req.Sort,
	}
	return s.SettingsRepo.CreatePaymentMethod(ctx, &m)
}

func (s *Service) UpdatePaymentMethod(ctx context.Context, op Operator, id int64, req dto.PaymentMethodReq) error {
	_, err := s.SettingsRepo.GetPaymentMethodByID(ctx, op.CompanyID, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errs.NotFound(errs.ErrPaymentMethodNotFound)
		}
		return err
	}
	return s.SettingsRepo.UpdatePaymentMethod(ctx, op.CompanyID, id, map[string]interface{}{
		"name": req.Name, "type": req.Type, "account_name": req.AccountName,
		"account_no": req.AccountNo, "qrcode": req.Qrcode, "status": req.Status,
		"sort": req.Sort, "updated_by": op.UserID,
	})
}

func (s *Service) DeletePaymentMethod(ctx context.Context, op Operator, id int64) error {
	return s.SettingsRepo.DeletePaymentMethod(ctx, op.CompanyID, id)
}

// -------- 操作日志 --------

func (s *Service) ListOperationLogs(ctx context.Context, op Operator, page, pageSize int) ([]model.SysOperationLog, int64, error) {
	return s.SettingsRepo.ListOperationLogs(ctx, op.CompanyID, page, pageSize)
}
