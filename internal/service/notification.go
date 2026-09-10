package service

import (
	"context"

	"photography-server/internal/enum"
	"photography-server/internal/model"
	"photography-server/internal/pkg/logger"
)

func (s *Service) ListNotifications(ctx context.Context, op Operator, page, pageSize int, unreadOnly bool) ([]model.SysNotification, int64, error) {
	return s.NotificationRepo.List(ctx, op.CompanyID, enum.NotificationReceiverStaff, op.UserID, page, pageSize, unreadOnly)
}

func (s *Service) UnreadNotificationCount(ctx context.Context, op Operator) (int64, error) {
	return s.NotificationRepo.UnreadCount(ctx, op.CompanyID, enum.NotificationReceiverStaff, op.UserID)
}

func (s *Service) MarkNotificationRead(ctx context.Context, op Operator, id int64) error {
	return s.NotificationRepo.MarkRead(ctx, op.CompanyID, enum.NotificationReceiverStaff, op.UserID, id)
}

func (s *Service) MarkAllNotificationsRead(ctx context.Context, op Operator) error {
	return s.NotificationRepo.MarkAllRead(ctx, op.CompanyID, enum.NotificationReceiverStaff, op.UserID)
}

// ---------------------------------------------------------------------
// 客户端（H5/小程序）站内通知：与员工通知共用 sys_notification 表，
// 以 receiver_type=2 隔离。员工与客户 ID 空间独立，不能只按 receiver_id 查。
// ---------------------------------------------------------------------

func (s *Service) ListClientNotifications(ctx context.Context, cu *ClientUser, page, pageSize int, unreadOnly bool) ([]model.SysNotification, int64, error) {
	return s.NotificationRepo.List(ctx, cu.CompanyID, enum.NotificationReceiverCustomer, cu.CustomerID, page, pageSize, unreadOnly)
}

func (s *Service) UnreadClientNotificationCount(ctx context.Context, cu *ClientUser) (int64, error) {
	return s.NotificationRepo.UnreadCount(ctx, cu.CompanyID, enum.NotificationReceiverCustomer, cu.CustomerID)
}

func (s *Service) MarkClientNotificationRead(ctx context.Context, cu *ClientUser, id int64) error {
	return s.NotificationRepo.MarkRead(ctx, cu.CompanyID, enum.NotificationReceiverCustomer, cu.CustomerID, id)
}

func (s *Service) MarkAllClientNotificationsRead(ctx context.Context, cu *ClientUser) error {
	return s.NotificationRepo.MarkAllRead(ctx, cu.CompanyID, enum.NotificationReceiverCustomer, cu.CustomerID)
}

// PushNotification 发送员工站内通知（供业务联动调用）
func (s *Service) PushNotification(ctx context.Context, op Operator, receiverID int64, ntype, title, content, bizType string, bizID int64) error {
	return s.pushNotification(ctx, op, enum.NotificationReceiverStaff, receiverID, ntype, title, content, bizType, bizID)
}

// NotifyClient 向客户推送站内通知（receiverID = crm_customer.id）。
// 与 NotifyStaff 同一约定：失败只记日志、绝不阻断主业务——
// 「预约已确认」这类通知发不出去，不能让确认动作本身失败。
func (s *Service) NotifyClient(ctx context.Context, op Operator, customerID int64, ntype, title, content, bizType string, bizID int64) {
	if err := s.pushNotification(ctx, op, enum.NotificationReceiverCustomer, customerID, ntype, title, content, bizType, bizID); err != nil {
		logger.Warnf("NotifyClient: 写入通知失败, customerID=%d, bizType=%s, bizID=%d, err=%v", customerID, bizType, bizID, err)
	}
}

// pushNotification 落库统一入口：receiverID 为 0 表示无接收人，直接跳过（不写孤儿通知）
func (s *Service) pushNotification(ctx context.Context, op Operator, receiverType int, receiverID int64, ntype, title, content, bizType string, bizID int64) error {
	if receiverID == 0 {
		return nil
	}
	return s.NotificationRepo.Create(ctx, &model.SysNotification{
		TenantBase: model.TenantBase{
			Base:      model.Base{CreatedBy: op.UserID, UpdatedBy: op.UserID},
			CompanyID: op.CompanyID,
		},
		ReceiverID:   receiverID,
		ReceiverType: receiverType,
		Type:         ntype,
		Title:        title,
		Content:      content,
		BizType:      bizType,
		BizID:        bizID,
		IsRead:       int(enum.NotificationUnread),
	})
}

// NotifyStaff 向工作室员工推送站内通知。
// receiverID > 0 时只发给该员工（如订单负责人）；否则广播给公司内全部启用员工
// （客户自助预约等场景订单尚无负责人，不广播就等于没人收到）。
// 通知失败只记日志，绝不阻断主业务（客户预约/退款申请必须成功）。
func (s *Service) NotifyStaff(ctx context.Context, op Operator, receiverID int64, ntype, title, content, bizType string, bizID int64) {
	receivers := make([]int64, 0, 8)
	if receiverID > 0 {
		receivers = append(receivers, receiverID)
	} else {
		ids, err := s.UserRepo.ListActiveIDs(ctx, op.CompanyID)
		if err != nil {
			logger.Warnf("NotifyStaff: 查询接收人失败, companyID=%d, err=%v", op.CompanyID, err)
			return
		}
		receivers = ids
	}
	for _, rid := range receivers {
		if err := s.PushNotification(ctx, op, rid, ntype, title, content, bizType, bizID); err != nil {
			logger.Warnf("NotifyStaff: 写入通知失败, receiverID=%d, bizType=%s, bizID=%d, err=%v", rid, bizType, bizID, err)
		}
	}
}
