package service

import (
	"photography-server/internal/model"
	"photography-server/internal/pkg/errs"
)

type CompanyUpdateReq struct {
	Name         string `json:"name"`
	Logo         string `json:"logo"`
	ContactName  string `json:"contact_name"`
	ContactPhone string `json:"contact_phone"`
	Address      string `json:"address"`
}

type Workspace struct {
	Company  model.SysCompany      `json:"company"`
	Stores   []model.SysStore      `json:"stores"`
	Roles    []model.SysRole       `json:"roles"`
	Payments []model.PaymentMethod `json:"payment_methods"`
}

func (s *Service) Workspace(op Operator) (*Workspace, error) {
	w := &Workspace{}
	if err := s.tenant(op).First(&w.Company, op.CompanyID).Error; err != nil {
		return nil, errs.NotFound("公司信息不存在")
	}
	var err error
	w.Stores, err = s.ListStores(op)
	if err != nil {
		return nil, err
	}
	w.Roles, err = s.ListRoles(op)
	if err != nil {
		return nil, err
	}
	w.Payments, err = s.ListPaymentMethods(op)
	if err != nil {
		return nil, err
	}
	return w, nil
}

func (s *Service) UpdateCompany(op Operator, req CompanyUpdateReq) error {
	return s.tenant(op).Model(&model.SysCompany{}).Where("id = ?", op.CompanyID).Updates(map[string]interface{}{
		"name": req.Name, "logo": req.Logo,
		"contact_name": req.ContactName, "contact_phone": req.ContactPhone,
		"address": req.Address, "updated_by": op.UserID,
	}).Error
}

// -------- 收款方式 --------

func (s *Service) ListPaymentMethods(op Operator) ([]model.PaymentMethod, error) {
	var list []model.PaymentMethod
	err := s.tenant(op).Order("sort ASC, id ASC").Find(&list).Error
	return list, err
}

func (s *Service) CreatePaymentMethod(op Operator, name, ptype, accountName, accountNo, qrcode string, status, sort int) error {
	m := model.PaymentMethod{
		TenantBase: model.TenantBase{
			Base:      model.Base{CreatedBy: op.UserID, UpdatedBy: op.UserID},
			CompanyID: op.CompanyID,
		},
		Name: name, Type: ptype, AccountName: accountName,
		AccountNo: accountNo, Qrcode: qrcode, Status: status, Sort: sort,
	}
	return s.tenant(op).Create(&m).Error
}

func (s *Service) UpdatePaymentMethod(op Operator, id int64, name, ptype, accountName, accountNo, qrcode string, status, sort int) error {
	var m model.PaymentMethod
	if err := s.tenant(op).First(&m, id).Error; err != nil {
		return errs.NotFound("收款方式不存在")
	}
	return s.tenant(op).Model(&m).Updates(map[string]interface{}{
		"name": name, "type": ptype, "account_name": accountName,
		"account_no": accountNo, "qrcode": qrcode, "status": status,
		"sort": sort, "updated_by": op.UserID,
	}).Error
}

func (s *Service) DeletePaymentMethod(op Operator, id int64) error {
	return s.tenant(op).Delete(&model.PaymentMethod{}, id).Error
}

// -------- 操作日志 --------

func (s *Service) ListOperationLogs(op Operator, page, pageSize int) ([]model.SysOperationLog, int64, error) {
	q := s.tenant(op)
	var total int64
	if err := q.Model(&model.SysOperationLog{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.SysOperationLog
	page, pageSize = normalizePage(page, pageSize)
	if err := q.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}
