package service

import (
	"context"
	"strings"

	"photography-server/internal/model"
	"photography-server/internal/pkg/errs"
	"photography-server/internal/presentation/dto"
)

// 反馈口径常量（与 model.SysFeedbackType 的取值、DDL 列宽对齐）
const (
	feedbackTypeBug    = "bug"    // 功能异常
	feedbackTypeAdvice = "advice" // 改进建议
	feedbackTypeOther  = "other"  // 其他
	feedbackMaxRunes   = 1000     // content varchar(1000)
	feedbackMaxImages  = 3        // 截图最多 3 张（列宽 1000 足够，限制来自产品口径）
)

// SubmitFeedback 员工提交意见反馈。
//
// 自助类能力：操作对象是提交人本人，不做角色校验（路由亦免权限点）。
// 服务端只做三件事：必填与长度校验、类型枚举收敛、截图数量截断。
// status 固定 1（待处理）—— 员工端不参与流转，避免自问自答式"已处理"。
func (s *Service) SubmitFeedback(ctx context.Context, op Operator, req dto.FeedbackSubmitReq) error {
	content := strings.TrimSpace(req.Content)
	if content == "" {
		return errs.BadRequest("请填写问题描述")
	}
	if len([]rune(content)) > feedbackMaxRunes {
		return errs.BadRequest("问题描述过长（最多 1000 字）")
	}
	typ := strings.TrimSpace(req.Type)
	if typ == "" {
		typ = feedbackTypeOther
	}
	switch typ {
	case feedbackTypeBug, feedbackTypeAdvice, feedbackTypeOther:
	default:
		return errs.BadRequest("问题类型不合法")
	}

	images := make([]string, 0, feedbackMaxImages)
	for _, u := range req.Images {
		if u = strings.TrimSpace(u); u != "" && len(images) < feedbackMaxImages {
			images = append(images, u)
		}
	}

	f := model.SysFeedback{
		TenantBase: model.TenantBase{
			Base:      model.Base{CreatedBy: op.UserID, UpdatedBy: op.UserID},
			CompanyID: op.CompanyID,
		},
		UserID:  op.UserID,
		Type:    typ,
		Status:  1, // 1-待处理
		Content: content,
		Images:  strings.Join(images, ","),
		Contact: strings.TrimSpace(req.Contact),
	}
	return s.FeedbackRepo.Create(ctx, &f)
}
