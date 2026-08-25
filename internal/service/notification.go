package service

import (
	"time"

	"photography-server/internal/model"
)

func (s *Service) ListNotifications(op Operator, page, pageSize int, unreadOnly bool) ([]model.SysNotification, int64, error) {
	q := s.tenant(op).Where("receiver_id = ?", op.UserID)
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

func (s *Service) UnreadNotificationCount(op Operator) (int64, error) {
	var count int64
	err := s.tenant(op).Model(&model.SysNotification{}).
		Where("receiver_id = ? AND is_read = ?", op.UserID, model.NotificationUnread).
		Count(&count).Error
	return count, err
}

func (s *Service) MarkNotificationRead(op Operator, id int64) error {
	now := time.Now().Format("2006-01-02 15:04:05")
	return s.tenant(op).Model(&model.SysNotification{}).
		Where("id = ? AND receiver_id = ?", id, op.UserID).
		Updates(map[string]interface{}{"is_read": model.NotificationRead, "read_at": now}).Error
}

func (s *Service) MarkAllNotificationsRead(op Operator) error {
	now := time.Now().Format("2006-01-02 15:04:05")
	return s.tenant(op).Model(&model.SysNotification{}).
		Where("receiver_id = ? AND is_read = ?", op.UserID, model.NotificationUnread).
		Updates(map[string]interface{}{"is_read": model.NotificationRead, "read_at": now}).Error
}

// PushNotification 发送站内通知（供业务联动调用）
func (s *Service) PushNotification(op Operator, receiverID int64, ntype, title, content, bizType string, bizID int64) error {
	if receiverID == 0 {
		return nil
	}
	return s.tenant(op).Create(&model.SysNotification{
		TenantBase: model.TenantBase{
			Base:      model.Base{CreatedBy: op.UserID, UpdatedBy: op.UserID},
			CompanyID: op.CompanyID,
		},
		ReceiverID: receiverID,
		Type:       ntype,
		Title:      title,
		Content:    content,
		BizType:    bizType,
		BizID:      bizID,
		IsRead:     model.NotificationUnread,
	}).Error
}
