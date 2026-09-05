package service

import (
	"context"

	"photography-server/internal/enum"
	"photography-server/internal/model"
)

func (s *Service) ListNotifications(ctx context.Context, op Operator, page, pageSize int, unreadOnly bool) ([]model.SysNotification, int64, error) {
	return s.NotificationRepo.List(ctx, op.CompanyID, op.UserID, page, pageSize, unreadOnly)
}

func (s *Service) UnreadNotificationCount(ctx context.Context, op Operator) (int64, error) {
	return s.NotificationRepo.UnreadCount(ctx, op.CompanyID, op.UserID)
}

func (s *Service) MarkNotificationRead(ctx context.Context, op Operator, id int64) error {
	return s.NotificationRepo.MarkRead(ctx, op.CompanyID, id)
}

func (s *Service) MarkAllNotificationsRead(ctx context.Context, op Operator) error {
	return s.NotificationRepo.MarkAllRead(ctx, op.CompanyID, op.UserID)
}

// PushNotification 发送站内通知（供业务联动调用）
func (s *Service) PushNotification(ctx context.Context, op Operator, receiverID int64, ntype, title, content, bizType string, bizID int64) error {
	if receiverID == 0 {
		return nil
	}
	return s.NotificationRepo.Create(ctx, &model.SysNotification{
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
		IsRead:     int(enum.NotificationUnread),
	})
}
