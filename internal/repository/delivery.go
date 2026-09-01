package repository

import (
	"gorm.io/gorm"

	"photography-server/internal/model"
)

type DeliveryRepo struct{}

func NewDeliveryRepo() *DeliveryRepo {
	return &DeliveryRepo{}
}

// GetByOrderID 根据订单ID查询交付单
func (r *DeliveryRepo) GetByOrderID(companyID, orderID int64) (*model.Delivery, error) {
	var d model.Delivery
	if err := tenant(companyID).Where("order_id = ?", orderID).First(&d).Error; err != nil {
		return nil, err
	}
	return &d, nil
}

// GetByID 根据 ID 查询交付单
func (r *DeliveryRepo) GetByID(companyID, id int64) (*model.Delivery, error) {
	var d model.Delivery
	if err := tenant(companyID).First(&d, id).Error; err != nil {
		return nil, err
	}
	return &d, nil
}

// GetOrCreateByOrder 获取或创建订单的交付单
func (r *DeliveryRepo) GetOrCreateByOrder(tx *gorm.DB, companyID, orderID int64, d *model.Delivery) (*model.Delivery, error) {
	if err := tx.Where("company_id = ? AND order_id = ?", companyID, orderID).First(d).Error; err == nil {
		return d, nil
	}
	if err := tx.Create(d).Error; err != nil {
		return nil, err
	}
	return d, nil
}

// Update 更新交付单
func (r *DeliveryRepo) Update(companyID, id int64, updates map[string]interface{}) error {
	return tenant(companyID).Model(&model.Delivery{}).Where("id = ?", id).Updates(updates).Error
}

// CreateItem 创建交付项
func (r *DeliveryRepo) CreateItem(tx *gorm.DB, item *model.DeliveryItem) error {
	return tx.Create(item).Error
}

// ListItems 查询交付项列表
func (r *DeliveryRepo) ListItems(companyID, deliveryID int64) ([]model.DeliveryItem, error) {
	var list []model.DeliveryItem
	err := tenant(companyID).Where("delivery_id = ?", deliveryID).Order("id ASC").Find(&list).Error
	return list, err
}

// CountItemsByKind 统计指定类型的交付项数量
func (r *DeliveryRepo) CountItemsByKind(companyID, deliveryID int64, kind string) (int64, error) {
	var count int64
	err := tenant(companyID).Model(&model.DeliveryItem{}).Where("delivery_id = ? AND kind = ?", deliveryID, kind).Count(&count).Error
	return count, err
}

// UpdateItemKind 更新交付项类型
func (r *DeliveryRepo) UpdateItemKind(companyID, deliveryID int64, itemIDs []int64, kind string) error {
	return tenant(companyID).Model(&model.DeliveryItem{}).Where("delivery_id = ? AND id IN ?", deliveryID, itemIDs).Update("kind", kind).Error
}
