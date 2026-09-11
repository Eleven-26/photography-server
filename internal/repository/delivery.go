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

// DeliveryListItem 交付单 + 订单快照（交付工作台看板用）。
// 订单编号/套餐名/拍摄日期取自订单表，避免前端为看板再查一次订单列表。
type DeliveryListItem struct {
	model.Delivery
	OrderCode   string `json:"order_code" gorm:"column:order_code"`
	PackageName string `json:"package_name" gorm:"column:package_name"`
	ShootDate   string `json:"shoot_date" gorm:"column:shoot_date"`
}

// List 交付单列表。stage<=0 不筛选阶段；keyword 命中客户姓名或订单编号。
// 注：这里用 Table + 别名手写租户/软删条件 —— Repo.tenant() 生成的裸
// `company_id = ?` 在 JOIN 下会因列名歧义报错。
func (r *DeliveryRepo) List(ctx context.Context, companyID int64, stage int, keyword string, page, pageSize int) ([]DeliveryListItem, int64, error) {
	base := func() *gorm.DB {
		db := r.conn().WithContext(ctx).
			Table("biz_delivery AS d").
			Joins("LEFT JOIN biz_order AS o ON o.id = d.order_id AND o.company_id = d.company_id").
			Where("d.company_id = ? AND d.deleted = 0", companyID)
		// 数据权限经已 JOIN 的订单别名传递（本表无 store_id）
		db = applyScope(db, opOf(ctx), scopeOrderJoined)
		if stage > 0 {
			db = db.Where("d.stage = ?", stage)
		}
		if keyword != "" {
			like := "%" + keyword + "%"
			db = db.Where("(d.customer_name LIKE ? OR o.code LIKE ?)", like, like)
		}
		return db
	}

	var total int64
	if err := base().Count(&total).Error; err != nil {
		return nil, 0, err
	}
	page, pageSize = normalizePage(page, pageSize)
	var list []DeliveryListItem
	if err := base().
		Select("d.*, COALESCE(o.code,'') AS order_code, COALESCE(o.package_name,'') AS package_name, COALESCE(o.shoot_date,'') AS shoot_date").
		Order("d.id DESC").Offset((page - 1) * pageSize).Limit(pageSize).
		Scan(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func (r *DeliveryRepo) Create(ctx context.Context, d *model.Delivery) error {
	return r.conn().WithContext(ctx).Create(d).Error
}

func (r *DeliveryRepo) Update(ctx context.Context, companyID, deliveryID int64, updates map[string]interface{}) error {
	return r.tenant(companyID).WithContext(ctx).Model(&model.Delivery{}).Where("id = ?", deliveryID).Updates(updates).Error
}

// CasConfirmExtra 客户确认加片（CAS）：仅当 extra_confirmed=0 时置 1，返回是否真正完成翻转。
// 用于防并发双击/重复请求把加片费重复累加进订单金额（#22）：调用方事务内先 CAS 抢占，
// 未抢到（RowsAffected=0）说明已被并发请求确认，不再累加金额。
func (r *DeliveryRepo) CasConfirmExtra(ctx context.Context, companyID, deliveryID int64) (bool, error) {
	res := r.tenant(companyID).WithContext(ctx).Model(&model.Delivery{}).
		Where("id = ? AND extra_confirmed = 0", deliveryID).
		Update("extra_confirmed", 1)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
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

// FeedbackListItem 客户修图反馈条目 + 交付单/订单编号（员工端「反馈整理」列表用）
type FeedbackListItem struct {
	model.DeliveryItem
	DeliveryCode string `json:"delivery_code" gorm:"column:delivery_code"`
	OrderCode    string `json:"order_code" gorm:"column:order_code"`
}

// ListFeedbackItems 客户反馈列表（报告 H11）。status<=0 表示全部；
// 只返回有反馈的明细（feedback_status > 0），待处理排在已处理之前。
// 注：与 List 同理，JOIN 下手写租户/软删条件，避免裸 company_id 列名歧义。
func (r *DeliveryRepo) ListFeedbackItems(ctx context.Context, companyID int64, status, page, pageSize int) ([]FeedbackListItem, int64, error) {
	base := func() *gorm.DB {
		db := r.conn().WithContext(ctx).
			Table("biz_delivery_item AS i").
			Joins("LEFT JOIN biz_delivery AS d ON d.id = i.delivery_id AND d.company_id = i.company_id").
			Joins("LEFT JOIN biz_order AS o ON o.id = i.order_id AND o.company_id = i.company_id").
			Where("i.company_id = ? AND i.deleted = 0 AND i.feedback_status > 0", companyID)
		// 数据权限经已 JOIN 的订单别名传递（本表无 store_id）
		db = applyScope(db, opOf(ctx), scopeOrderJoined)
		if status > 0 {
			db = db.Where("i.feedback_status = ?", status)
		}
		return db
	}

	var total int64
	if err := base().Count(&total).Error; err != nil {
		return nil, 0, err
	}
	page, pageSize = normalizePage(page, pageSize)
	var list []FeedbackListItem
	if err := base().
		Select("i.*, COALESCE(d.code,'') AS delivery_code, COALESCE(o.code,'') AS order_code").
		Order("i.feedback_status ASC, i.id DESC").Offset((page - 1) * pageSize).Limit(pageSize).
		Scan(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}
