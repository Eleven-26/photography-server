package repository

import (
	"context"

	"gorm.io/gorm"

	"photography-server/internal/enum"
	"photography-server/internal/model"
)

type NotificationRepo struct {
	Repo
}

// WithTx 返回绑定到指定事务连接的副本，事务内的所有写操作将复用该连接，
// 保证跨多张表的写入原子性（失败自动回滚）。
func (r *NotificationRepo) WithTx(tx *gorm.DB) *NotificationRepo {
	return &NotificationRepo{Repo: Repo{db: tx}}
}

func NewNotificationRepo() *NotificationRepo { return &NotificationRepo{} }

// List 列出接收人的通知。receiverType 区分员工/客户（见 enum.NotificationReceiver）：
// 两者 ID 空间独立，只用 receiver_id 过滤会让「客户 5」看到「员工 5」的通知。
func (r *NotificationRepo) List(ctx context.Context, companyID int64, receiverType int, receiverID int64, page, pageSize int, onlyUnread bool) ([]model.SysNotification, int64, error) {
	q := r.tenant(companyID).WithContext(ctx).
		Where("receiver_type = ? AND receiver_id = ?", receiverType, receiverID)
	if onlyUnread {
		q = q.Where("is_read = ?", int(enum.NotificationUnread))
	}
	var total int64
	if err := q.Model(&model.SysNotification{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.SysNotification
	page, pageSize = normalizePage(page, pageSize)
	if err := q.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func (r *NotificationRepo) UnreadCount(ctx context.Context, companyID int64, receiverType int, receiverID int64) (int64, error) {
	var count int64
	err := r.tenant(companyID).WithContext(ctx).Model(&model.SysNotification{}).
		Where("receiver_type = ? AND receiver_id = ? AND is_read = ?", receiverType, receiverID, int(enum.NotificationUnread)).
		Count(&count).Error
	return count, err
}

// MarkRead 标记单条已读。必须同时校验接收人归属：仅凭 company_id + id 时，
// 同公司内任意员工/客户都能把别人的通知标记为已读。
func (r *NotificationRepo) MarkRead(ctx context.Context, companyID int64, receiverType int, receiverID, notificationID int64) error {
	return r.tenant(companyID).WithContext(ctx).Model(&model.SysNotification{}).
		Where("id = ? AND receiver_type = ? AND receiver_id = ?", notificationID, receiverType, receiverID).
		Update("is_read", int(enum.NotificationRead)).Error
}

func (r *NotificationRepo) MarkAllRead(ctx context.Context, companyID int64, receiverType int, receiverID int64) error {
	return r.tenant(companyID).WithContext(ctx).Model(&model.SysNotification{}).
		Where("receiver_type = ? AND receiver_id = ? AND is_read = ?", receiverType, receiverID, int(enum.NotificationUnread)).
		Update("is_read", int(enum.NotificationRead)).Error
}

func (r *NotificationRepo) Create(ctx context.Context, n *model.SysNotification) error {
	return r.conn().WithContext(ctx).Create(n).Error
}
