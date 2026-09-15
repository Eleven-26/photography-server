package repository

import (
	"context"

	"photography-server/internal/model"
)

// ---------------------------------------------------------------------
// 客户中心 · 服务关系候选（H5 定制需求页「选择门店 → 选择摄影师」）
//
// 候选口径：客户**曾下过单或提过定制需求**的门店与摄影师 ——
// 不下发全量员工名册：客户只该看到服务过自己的人（既避免随手指定一位陌生摄影师，
// 也避免把工作室的组织结构暴露到客户端）。
//
// 两张来源表字段同构（biz_order / biz_custom_request 均有 store_id + photographer_id），
// 故共用 CustomerStaffPair，各查一次后在 service 层按 (store_id, photographer_id) 去重合并。
//
// ⚠️ biz_custom_request.photographer_id 是 2026-09-15 新增列
//    （docs/sql/增量/upgrade_custom_request_photographer_20260915.sql）：此前提交的历史
//    定制需求该列为 0，视为「未指定摄影师」，只会贡献 store_id，不会贡献摄影师候选。
// ---------------------------------------------------------------------

// CustomerStaffPair 客户历史关联的「门店 + 摄影师」组合（已去重）。
// PhotographerID = 0 表示该次关联未指定摄影师（订单未指派 / 需求未选择）。
type CustomerStaffPair struct {
	StoreID        int64 `gorm:"column:store_id"`
	PhotographerID int64 `gorm:"column:photographer_id"`
}

// ListCustomerStaffPairs 客户在**订单**里关联过的门店+摄影师组合（去重）。
//
// 注：这里手写 company_id / deleted 条件 —— Table() 是原生查询，不经过模型作用域，
// 软删标识不会被自动追加（同 ReviewRepo.ListByCustomer 的做法）。
func (r *OrderRepo) ListCustomerStaffPairs(ctx context.Context, companyID, customerID int64) ([]CustomerStaffPair, error) {
	var list []CustomerStaffPair
	err := r.conn().WithContext(ctx).
		Table("biz_order").
		Select("DISTINCT store_id, photographer_id").
		Where("company_id = ? AND customer_id = ? AND deleted = 0", companyID, customerID).
		Scan(&list).Error
	return list, err
}

// ListCustomerStaffPairs 客户在**定制需求**里关联过的门店+摄影师组合（去重）。
func (r *CustomRequestRepo) ListCustomerStaffPairs(ctx context.Context, companyID, customerID int64) ([]CustomerStaffPair, error) {
	var list []CustomerStaffPair
	err := r.conn().WithContext(ctx).
		Table("biz_custom_request").
		Select("DISTINCT store_id, photographer_id").
		Where("company_id = ? AND customer_id = ? AND deleted = 0", companyID, customerID).
		Scan(&list).Error
	return list, err
}

// ListByIDs 按 ID 批量取员工（同租户内）。用于把候选摄影师 ID 换成姓名 / 头像 / 所属门店。
// 不过滤 status —— 是否只保留启用中的员工由调用方决定（客户端候选只留启用的，见
// service.ClientPhotographerOptions：停用员工接不了单，给了客户也会被提交接口拒绝）。
func (r *UserRepo) ListByIDs(ctx context.Context, companyID int64, ids []int64) ([]model.SysUser, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var list []model.SysUser
	err := r.tenant(companyID).WithContext(ctx).Where("id IN ?", ids).Find(&list).Error
	return list, err
}
