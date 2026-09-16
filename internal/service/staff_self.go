package service

import (
	"context"
	"strings"

	"photography-server/internal/domain"
	"photography-server/internal/pkg/errs"
)

// smsSceneChangeMobile 换绑手机号的短信场景。
// 与登录场景（"login"）刻意隔离：登录验证码不能拿来换绑手机号。
// 否则只要拿到一个登录码，就能把账号手机号改到别人名下（验证码降级为万能凭证）。
const smsSceneChangeMobile = "change_mobile"

// SendStaffMobileCode 向**当前绑定手机号**发送换绑验证码。
//
// 为什么发旧号而不是新号：验证码的作用是证明"操作人 = 账号持有人"。
// 新号此刻尚未与账号建立关系，发到新号等于向任意号码索要凭证 —— 无验证意义。
// 因此先验旧号换绑资格，再在 ChangeStaffMobile 里落新号。
func (s *Service) SendStaffMobileCode(ctx context.Context, op Operator) error {
	u, err := s.AuthRepo.GetByID(ctx, op.CompanyID, op.UserID)
	if err != nil {
		return errs.NotFound(errs.ErrUserNotFound)
	}
	mobile := strings.TrimSpace(u.Mobile)
	if mobile == "" {
		return errs.BadRequest(errs.ErrStaffMobileUnbound)
	}
	return s.SendSmsCode(ctx, smsSceneChangeMobile, mobile)
}

// ChangeStaffMobile 校验换绑验证码并更新本人手机号。
//
// 顺序：格式校验 → 取当前号 → 与当前号比对 → **消费验证码** → 占用校验 → 落库。
// 验证码先消费再去查占用：验证码是一次性凭证，把它排在"可能因占用而失败"的检查之后，
// 会让用户在号码已被占用时白白消耗一次验证码，得重新发码再来一遍。
func (s *Service) ChangeStaffMobile(ctx context.Context, op Operator, code, newMobile string) error {
	newMobile = strings.TrimSpace(newMobile)
	if !domain.IsMobile(newMobile) {
		return errs.BadRequest(errs.ErrMobileFormatWrong)
	}
	u, err := s.AuthRepo.GetByID(ctx, op.CompanyID, op.UserID)
	if err != nil {
		return errs.NotFound(errs.ErrUserNotFound)
	}
	cur := strings.TrimSpace(u.Mobile)
	if cur == "" {
		return errs.BadRequest(errs.ErrStaffMobileUnbound)
	}
	if cur == newMobile {
		return errs.BadRequest(errs.ErrMobileUnchanged)
	}
	if err := s.verifySmsCode(ctx, smsSceneChangeMobile, cur, code); err != nil {
		return err
	}
	// 占用校验：同号多租户时 GetByMobile 取最早注册者，故仅当命中且非本人时拒绝
	if other, oerr := s.AuthRepo.GetByMobile(ctx, newMobile); oerr == nil && other != nil && other.ID != op.UserID {
		return errs.Conflict(errs.ErrMobileTaken)
	}
	return s.AuthRepo.UpdateMobile(ctx, op.UserID, newMobile)
}
