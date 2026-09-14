package repository

import (
	"context"

	"photography-server/internal/model"
)

// FeedbackRepo 员工意见反馈（sys_feedback）。
// 只有一条写路径：员工提交。反馈的流转（status / handled_at / remark）由管理后台另行维护，
// 员工端不参与，故此处不提供 Update —— 避免"能改自己反馈状态"的口子。
type FeedbackRepo struct {
	Repo
}

func NewFeedbackRepo() *FeedbackRepo { return &FeedbackRepo{} }

// Create 落库一条反馈（company_id / user_id 由 service 在 model 上填充）
func (r *FeedbackRepo) Create(ctx context.Context, f *model.SysFeedback) error {
	return r.conn().WithContext(ctx).Create(f).Error
}
