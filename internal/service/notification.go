package service

import (
	"time"

	"photography-server/internal/model"
)

func (s *Service) ListNotifications(op Operator, page, pageSize int, unreadOnly bool) ([]model.SysNotification, int64, error) {
	return s.NotificationRepo.List(op.CompanyID, op.UserID, page, pageSize, unreadOnly)
}

func (s *Service) UnreadNotificationCount(op Operator) (int64, error) {
	return s.NotificationRepo.UnreadCount(op.CompanyID, op.UserID)
}

func (s *Service) MarkNotificationRead(op Operator, id int64) error {
	now := time.Now().Format("2006-01-02 15:04:05")
	return s.NotificationRepo.MarkRead(op.CompanyID, op.UserID, id, now)
}

func (s *Service) MarkAllNotificationsRead(op Operator) error {
	now := time.Now().Format("2006-01-02 15:04:05")
	return s.NotificationRepo.MarkAllRead(op.CompanyID, op.UserID, now)
}

// PushNotification 发送站内通知（供业务联动调用）
func (s *Service) PushNotification(op Operator, receiverID int64, ntype, title, content, bizType string, bizID int64) error {
	if receiverID == 0 {
		return nil
	}
	return s.NotificationRepo.Create(&model.SysNotification{
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
	})
}
