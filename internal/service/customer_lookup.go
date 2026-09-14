package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	"photography-server/internal/domain"
	"photography-server/internal/enum"
	"photography-server/internal/model"
	"photography-server/internal/pkg/errs"
	"photography-server/internal/repository"
)

// CustomerSourceLead 线索链路自动建档时「客户来源」的兜底值：线索自身没填来源时使用。
const CustomerSourceLead = "线索"

// FindOrCreateCustomerByMobile 按手机号在客户表查档，不存在则以 name/mobile 建档后返回（默认连接）。
//
// 场景：客户从「非套餐」入口进来（新增线索、H5 定制需求提交、定制需求转订单），
// 这些链路此前只把 name/mobile 写进业务表，客户主体始终缺失，
// 导致订单/交付/通知无法按客户聚合。这里统一做「查档 → 建档」。
//
// 约定：
//   - 手机号视为租户内客户唯一键（与客户端验证码登录 GetByMobile 同口径）；
//     crm_customer.mobile 只有普通索引、无唯一约束，去重完全依赖本函数，勿绕过；
//   - mobile 为空返回 (nil, nil)，由调用方按「无客户」处理 —— 无手机号无法去重，不建档；
//   - 数据库故障（非 ErrRecordNotFound）直接抛错，不误判为「未建档」而重复建档。
func (s *Service) FindOrCreateCustomerByMobile(ctx context.Context, companyID int64, name, mobile, source string) (*model.Customer, error) {
	return s.findOrCreateCustomerByMobile(ctx, s.CustomerRepo, companyID, name, mobile, source)
}

// findOrCreateCustomerByMobile 与 FindOrCreateCustomerByMobile 同语义，
// 但允许传入事务绑定的仓储副本：需要「客户建档 + 业务写入」原子落库的调用方
// （如新增线索）传 s.CustomerRepo.WithTx(tx)，任一步失败整体回滚，
// 避免留下「线索建了、客户没建」或反之的半成品数据。
func (s *Service) findOrCreateCustomerByMobile(ctx context.Context, repo *repository.CustomerRepo, companyID int64, name, mobile, source string) (*model.Customer, error) {
	mobile = strings.TrimSpace(mobile)
	if mobile == "" {
		return nil, nil
	}

	c, err := repo.GetByMobile(ctx, companyID, mobile)
	if err == nil {
		return c, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errs.Internal("")
	}

	if strings.TrimSpace(name) == "" {
		name = maskMobile(mobile)
	}
	c = &model.Customer{
		TenantBase: model.TenantBase{
			Base:      model.Base{CreatedAt: time.Now(), UpdatedAt: time.Now()},
			CompanyID: companyID,
		},
		Code:   domain.GenCode("CU"),
		Name:   name,
		Mobile: mobile,
		Source: source,
		Status: enum.CustomerStatusActive,
	}
	if err := repo.Create(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}
