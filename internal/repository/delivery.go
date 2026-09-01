package repository

import (
	"photography-server/internal/infrastructure"
	"photography-server/internal/model"
)

type DeliveryRepo struct{}

func NewDeliveryRepo() *DeliveryRepo { return &DeliveryRepo{} }

func (r *DeliveryRepo) GetByID(companyID, deliveryID int64) (*model.Delivery, error) {
	var d model.Delivery
	if err := tenant(companyID).First(&d, deliveryID).Error; err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *DeliveryRepo) GetByOrderID(companyID, orderID int64) (*model.Delivery, error) {
	var d model.Delivery
	if err := tenant(companyID).Where("order_id = ?", orderID).First(&d).Error; err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *DeliveryRepo) Create(d *model.Delivery) error {
	return infrastructure.MySQL().Create(d).Error
}

func (r *DeliveryRepo) Update(companyID, deliveryID int64, updates map[string]interface{}) error {
	return tenant(companyID).Model(&model.Delivery{}).Where("id = ?", deliveryID).Updates(updates).Error
}

func (r *DeliveryRepo) CreateItem(item *model.DeliveryItem) error {
	return infrastructure.MySQL().Create(item).Error
}

func (r *DeliveryRepo) UpdateItemKind(companyID, itemID int64, kind string) error {
	return tenant(companyID).Model(&model.DeliveryItem{}).Where("id = ?", itemID).Update("kind", kind).Error
}

func (r *DeliveryRepo) ListItems(companyID, deliveryID int64) ([]model.DeliveryItem, error) {
	var list []model.DeliveryItem
	err := tenant(companyID).Where("delivery_id = ?", deliveryID).Order("id ASC").Find(&list).Error
	return list, err
}
