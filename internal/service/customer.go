package service

import (
	"gorm.io/gorm"

	"photography-server/internal/model"
	"photography-server/internal/pkg/errs"
)

type CustomerCreateReq struct {
	StoreID  int64  `json:"store_id"`
	Name     string `json:"name" binding:"required"`
	Mobile   string `json:"mobile"`
	Wechat   string `json:"wechat"`
	Gender   string `json:"gender"`
	Birthday string `json:"birthday"`
	Level    string `json:"level"`
	Source   string `json:"source"`
	Tags     string `json:"tags"`
	Status   string `json:"status"`
	Remark   string `json:"remark"`
	Avatar   string `json:"avatar"`
}

type CustomerUpdateReq struct {
	StoreID  int64  `json:"store_id"`
	Name     string `json:"name"`
	Mobile   string `json:"mobile"`
	Wechat   string `json:"wechat"`
	Gender   string `json:"gender"`
	Birthday string `json:"birthday"`
	Level    string `json:"level"`
	Source   string `json:"source"`
	Tags     string `json:"tags"`
	Status   string `json:"status"`
	Remark   string `json:"remark"`
	Avatar   string `json:"avatar"`
}

func (s *Service) ListCustomers(op Operator, page, pageSize int, keyword, status, level string) ([]model.Customer, int64, error) {
	q := s.tenant(op)
	if keyword != "" {
		kw := "%" + keyword + "%"
		q = q.Where("name LIKE ? OR mobile LIKE ? OR wechat LIKE ? OR code LIKE ?", kw, kw, kw, kw)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if level != "" {
		q = q.Where("level = ?", level)
	}
	var total int64
	if err := q.Model(&model.Customer{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.Customer
	page, pageSize = normalizePage(page, pageSize)
	if err := q.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func (s *Service) GetCustomer(op Operator, id int64) (*model.Customer, error) {
	var c model.Customer
	if err := s.tenant(op).First(&c, id).Error; err != nil {
		return nil, errs.NotFound("客户不存在")
	}
	return &c, nil
}

// createCustomerTx 在指定事务内创建客户（供线索转客户等场景复用）
func (s *Service) createCustomerTx(tx *gorm.DB, op Operator, req CustomerCreateReq) (*model.Customer, error) {
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
	if err := tx.Create(&c).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *Service) CreateCustomer(op Operator, req CustomerCreateReq) (*model.Customer, error) {
	return s.createCustomerTx(s.tenant(op), op, req)
}

func (s *Service) UpdateCustomer(op Operator, id int64, req CustomerUpdateReq) error {
	var c model.Customer
	if err := s.tenant(op).First(&c, id).Error; err != nil {
		return errs.NotFound("客户不存在")
	}
	return s.tenant(op).Model(&c).Updates(map[string]interface{}{
		"store_id": req.StoreID, "name": req.Name, "mobile": req.Mobile,
		"wechat": req.Wechat, "gender": req.Gender, "birthday": req.Birthday,
		"level": req.Level, "source": req.Source, "tags": req.Tags,
		"status": req.Status, "remark": req.Remark, "avatar": req.Avatar,
		"updated_by": op.UserID,
	}).Error
}

func (s *Service) DeleteCustomer(op Operator, id int64) error {
	return s.tenant(op).Delete(&model.Customer{}, id).Error
}

type CustomerStats struct {
	Total        int64 `json:"total"`
	Active       int64 `json:"active"`
	GoldUp       int64 `json:"gold_up"`
	NewThisMonth int64 `json:"new_this_month"`
}

func (s *Service) CustomerStats(op Operator) (*CustomerStats, error) {
	st := &CustomerStats{}
	q := s.tenant(op)
	if err := q.Model(&model.Customer{}).Count(&st.Total).Error; err != nil {
		return nil, err
	}
	if err := q.Model(&model.Customer{}).Where("status = ?", model.CustomerStatusActive).Count(&st.Active).Error; err != nil {
		return nil, err
	}
	if err := q.Model(&model.Customer{}).Where("level IN ?", []string{model.CustomerLevelGold, model.CustomerLevelPlatinum, model.CustomerLevelDiamond}).Count(&st.GoldUp).Error; err != nil {
		return nil, err
	}
	return st, nil
}
