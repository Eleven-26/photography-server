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
)

// FindOrCreateCustomerByMobile 按手机号在客户表查档，不存在则以 name/mobile 建档后返回。
//
// 场景：客户从「非套餐」入口进来（H5 定制需求提交、定制需求转订单），
// 这些链路此前只把 name/mobile 写进业务表，客户主体始终缺失，
// 导致订单/交付/通知无法按客户聚合。这里统一做「查档 → 建档」。
//
// 约定：
//   - 手机号视为租户内客户唯一键（与客户端验证码登录 GetByMobile 同口径）；
//   - mobile 为空返回 (nil, nil)，由调用方按「无客户」处理 —— 无手机号无法去重，不建档；
//   - 数据库故障（非 ErrRecordNotFound）直接抛错，不误判为「未建档」而重复建档。
func (s *Service) FindOrCreateCustomerByMobile(ctx context.Context, companyID int64, name, mobile, source string) (*model.Customer, error) {
	mobile = strings.TrimSpace(mobile)
	if mobile == "" {
		return nil, nil
	}

	c, err := s.CustomerRepo.GetByMobile(ctx, companyID, mobile)
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
	if err := s.CustomerRepo.Create(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}
