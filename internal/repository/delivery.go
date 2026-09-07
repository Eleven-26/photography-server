package repository

import (
	"context"

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

func (r *DeliveryRepo) GetByID(ctx context.Context, companyID, deliveryID int64) (*model.Delivery, error) {
	var d model.Delivery
	if err := r.tenant(companyID).WithContext(ctx).First(&d, deliveryID).Error; err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *DeliveryRepo) GetByOrderID(ctx context.Context, companyID, orderID int64) (*model.Delivery, error) {
	var d model.Delivery
	if err := r.tenant(companyID).WithContext(ctx).Where("order_id = ?", orderID).First(&d).Error; err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *DeliveryRepo) Create(ctx context.Context, d *model.Delivery) error {
	return r.conn().WithContext(ctx).Create(d).Error
}

func (r *DeliveryRepo) Update(ctx context.Context, companyID, deliveryID int64, updates map[string]interface{}) error {
	return r.tenant(companyID).WithContext(ctx).Model(&model.Delivery{}).Where("id = ?", deliveryID).Updates(updates).Error
}

func (r *DeliveryRepo) CreateItem(ctx context.Context, item *model.DeliveryItem) error {
	return r.conn().WithContext(ctx).Create(item).Error
}

func (r *DeliveryRepo) UpdateItemKind(ctx context.Context, companyID, itemID int64, kind string) error {
	return r.tenant(companyID).WithContext(ctx).Model(&model.DeliveryItem{}).Where("id = ?", itemID).Update("kind", kind).Error
}

// UpdateItem 更新交付明细字段（客户选片/修图反馈等）
func (r *DeliveryRepo) UpdateItem(ctx context.Context, companyID, itemID int64, updates map[string]interface{}) error {
	return r.tenant(companyID).WithContext(ctx).Model(&model.DeliveryItem{}).
		Where("id = ?", itemID).Updates(updates).Error
}

// FirstItem 按 ID 取单条交付明细（租户过滤）
func (r *DeliveryRepo) FirstItem(ctx context.Context, companyID, itemID int64, out *model.DeliveryItem) error {
	return r.tenant(companyID).WithContext(ctx).First(out, itemID).Error
}

func (r *DeliveryRepo) ListItems(ctx context.Context, companyID, deliveryID int64) ([]model.DeliveryItem, error) {
	var list []model.DeliveryItem
	err := r.tenant(companyID).WithContext(ctx).Where("delivery_id = ?", deliveryID).Order("id ASC").Find(&list).Error
	return list, err
}
