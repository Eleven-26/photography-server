package service

import (
	"context"

	"photography-server/internal/presentation/dto"
)

// Overview 工作台概览。repository 负责取数，本层映射为对外契约（dto），
// 使仓储/数据库结构调整不会直接泄漏到 API 响应。
func (s *Service) Overview(ctx context.Context, op Operator) (*dto.DashboardOverviewResp, error) {
	ov, err := s.DashboardRepo.GetOverview(ctx, op.CompanyID, op.UserID)
	if err != nil {
		return nil, err
	}

	resp := &dto.DashboardOverviewResp{
		TodayOrders:       ov.TodayOrders,
		TodayAmount:       ov.TodayAmount,
		MonthOrders:       ov.MonthOrders,
		MonthAmount:       ov.MonthAmount,
		PendingPayments:   ov.PendingPayments,
		PendingDeliveries: ov.PendingDeliveries,
		NewLeads:          ov.NewLeads,
		OverdueLeads:      ov.OverdueLeads,
		PendingDeposit:    ov.PendingDeposit,
		PendingRetouch:    ov.PendingRetouch,
		UpcomingShoots:    ov.UpcomingShoots,
		TodayConfirmed:    ov.TodayConfirmed,
		TodayPending:      ov.TodayPending,
		UnreadNotify:      ov.UnreadNotify,
		MonthLeads:        ov.MonthLeads,
		MonthDealRate:     ov.MonthDealRate,
		AvailableSlots:    ov.AvailableSlots,
	}

	// 列表型字段显式转换：dto 与仓储结构各自演进，避免字段名漂移直接穿透到 API。
	for _, ts := range ov.TodayShoots {
		resp.TodayShoots = append(resp.TodayShoots, dto.TodayShoot{
			ID:           ts.ID,
			Code:         ts.Code,
			CustomerName: ts.CustomerName,
			PackageName:  ts.PackageName,
			ShootTime:    ts.ShootTime,
			ShootAddress: ts.ShootAddress,
			Photographer: ts.Photographer,
			Status:       ts.Status,
		})
	}
	for _, td := range ov.TodoItems {
		resp.TodoItems = append(resp.TodoItems, dto.TodoItem{
			Key:   td.Key,
			Label: td.Label,
			Count: td.Count,
			Route: td.Route,
			Tone:  td.Tone,
		})
	}

	return resp, nil
}
