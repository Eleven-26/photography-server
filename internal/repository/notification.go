package repository

import (
	"photography-server/internal/model"
)

type NotificationRepo struct{}

func NewNotificationRepo() *NotificationRepo {
	return &NotificationRepo{}
}

// List 通知列表（分页 + 是否未读筛选）
func (r *NotificationRepo) List(companyID, userID int64, page, pageSize int, unreadOnly bool) ([]model.SysNotification, int64, error) {
	q := tenant(companyID).Where("receiver_id = ?", userID)
	if unreadOnly {
		q = q.Where("is_read = ?", model.NotificationUnread)
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

// UnreadCount 未读通知数
func (r *NotificationRepo) UnreadCount(companyID, userID int64) (int64, error) {
	var count int64
	err := tenant(companyID).Model(&model.SysNotification{}).
		Where("receiver_id = ? AND is_read = ?", userID, model.NotificationUnread).
		Count(&count).Error
	return count, err
}

// MarkRead 标记单条通知已读
func (r *NotificationRepo) MarkRead(companyID, userID, id int64, readAt string) error {
	return tenant(companyID).Model(&model.SysNotification{}).
		Where("id = ? AND receiver_id = ?", id, userID).
		Updates(map[string]interface{}{"is_read": model.NotificationRead, "read_at": readAt}).Error
}

// MarkAllRead 标记全部未读通知已读
func (r *NotificationRepo) MarkAllRead(companyID, userID int64, readAt string) error {
	return tenant(companyID).Model(&model.SysNotification{}).
		Where("receiver_id = ? AND is_read = ?", userID, model.NotificationUnread).
		Updates(map[string]interface{}{"is_read": model.NotificationRead, "read_at": readAt}).Error
}

// Create 创建通知
func (r *NotificationRepo) Create(n *model.SysNotification) error {
	return tenant(n.CompanyID).Create(n).Error
}
