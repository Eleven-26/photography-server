package repository

import (
	"gorm.io/gorm"
	"photography-server/internal/model"
)

type DeliveryRepo struct {
	Repo
}

// WithTx 返回绑定到指定事务连接的副本，事务内的所有写操作将复用该连接，
// 保证跨多张表的写入原子性（失败自动回滚）。
func (r *DeliveryRepo) WithTx(tx *gorm.DB) *DeliveryRepo {
	return &DeliveryRepo{Repo: Repo{db: tx}}
}

func NewDeliveryRepo() *DeliveryRepo { return &DeliveryRepo{} }

func (r *DeliveryRepo) GetByID(companyID, deliveryID int64) (*model.Delivery, error) {
	var d model.Delivery
	if err := r.tenant(companyID).First(&d, deliveryID).Error; err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *DeliveryRepo) GetByOrderID(companyID, orderID int64) (*model.Delivery, error) {
	var d model.Delivery
	if err := r.tenant(companyID).Where("order_id = ?", orderID).First(&d).Error; err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *DeliveryRepo) Create(d *model.Delivery) error {
	return r.conn().Create(d).Error
}

func (r *DeliveryRepo) Update(companyID, deliveryID int64, updates map[string]interface{}) error {
	return r.tenant(companyID).Model(&model.Delivery{}).Where("id = ?", deliveryID).Updates(updates).Error
}

func (r *DeliveryRepo) CreateItem(item *model.DeliveryItem) error {
	return r.conn().Create(item).Error
}

func (r *DeliveryRepo) UpdateItemKind(companyID, itemID int64, kind string) error {
	return r.tenant(companyID).Model(&model.DeliveryItem{}).Where("id = ?", itemID).Update("kind", kind).Error
}

func (r *DeliveryRepo) ListItems(companyID, deliveryID int64) ([]model.DeliveryItem, error) {
	var list []model.DeliveryItem
	err := r.tenant(companyID).Where("delivery_id = ?", deliveryID).Order("id ASC").Find(&list).Error
	return list, err
}
