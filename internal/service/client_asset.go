package service

import (
	"context"

	"photography-server/internal/enum"
	"photography-server/internal/model"
	"photography-server/internal/pkg/errs"
)

// ---------------------------------------------------------------------
// 客户侧公开作品集（报告 H5：C24 作品列表、预约主页作品展示）。
// 可见范围由服务端写死（已发布 + 公开），不接受客户端传 status/visibility；
// 未公开作品对客户一律「不存在」，防止用 ID 遍历窥探内部作品。
// ---------------------------------------------------------------------

// ClientAssets 公开作品列表（精选置顶）
func (s *Service) ClientAssets(ctx context.Context, companyID int64, page, pageSize int, category string, featuredOnly bool) ([]model.Asset, int64, error) {
	return s.AssetRepo.ListPublic(ctx, companyID, page, pageSize, category, featuredOnly)
}

// ClientAssetDetail 公开作品详情（浏览数 +1）
func (s *Service) ClientAssetDetail(ctx context.Context, companyID, id int64) (*model.Asset, error) {
	a, err := s.AssetRepo.GetByID(ctx, companyID, id)
	if err != nil {
		return nil, errs.NotFound("作品不存在或未公开")
	}
	if a.Status != enum.AssetStatusPublished || a.Visibility != enum.AssetVisibilityPublic {
		return nil, errs.NotFound("作品不存在或未公开")
	}
	// 浏览数自增失败不影响详情返回（计数属旁路数据）
	if err := s.AssetRepo.IncrViewCount(ctx, companyID, id); err == nil {
		a.ViewCount++
	}
	return a, nil
}
