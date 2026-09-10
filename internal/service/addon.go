package service

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"photography-server/internal/domain"
	"photography-server/internal/enum"
	"photography-server/internal/model"
	"photography-server/internal/pkg/errs"
	"photography-server/internal/presentation/dto"
	"photography-server/internal/repository"
)

// 订单加项（妆造 / 时效 / 服务 / 精修等）。
//
// 资金口径：加项金额计入订单总额与尾款，因此任何增删改都必须在**同一事务**内
// 重算订单的 addon_amount / deposit_amt / final_amt / total_amt，并写操作日志。
// 否则会出现"加项改了、订单总额没变"的资金口径漂移（对账、退款试算全部失真）。

// ListOrderAddons 订单加项列表
func (s *Service) ListOrderAddons(ctx context.Context, op Operator, orderID int64) ([]model.OrderAddon, error) {
	if _, err := s.OrderRepo.GetByID(ctx, op.CompanyID, orderID); err != nil {
		return nil, errs.NotFound(errs.ErrOrderNotFound)
	}
	return s.AddonRepo.ListByOrder(ctx, op.CompanyID, orderID)
}

// CreateOrderAddon 新增加项：写加项 + 重算订单金额（同事务）
func (s *Service) CreateOrderAddon(ctx context.Context, op Operator, orderID int64, req dto.OrderAddonReq) (*model.OrderAddon, error) {
	o, err := s.ensureOrderEditable(ctx, op.CompanyID, orderID)
	if err != nil {
		return nil, err
	}
	qty := req.Qty
	if qty <= 0 {
		qty = 1
	}
	list := []model.OrderAddon{{
		TenantBase: model.TenantBase{
			Base:      model.Base{CreatedBy: op.UserID, UpdatedBy: op.UserID},
			CompanyID: op.CompanyID,
		},
		OrderID:   orderID,
		Name:      req.Name,
		Category:  req.Category,
		Price:     req.Price,
		Qty:       qty,
		Amount:    domain.Round2(req.Price * float64(qty)),
		Confirmed: req.Confirmed,
		Remark:    req.Remark,
	}}
	if err := repository.Tx(func(tx *gorm.DB) error {
		if err := s.AddonRepo.WithTx(tx).CreateBatch(ctx, list); err != nil {
			return err
		}
		return s.recalcOrderAmountsTx(ctx, tx, o, "新增加项 "+list[0].Name, op)
	}); err != nil {
		return nil, err
	}
	return &list[0], nil
}

// UpdateOrderAddon 修改加项：更新加项 + 重算订单金额（同事务）
func (s *Service) UpdateOrderAddon(ctx context.Context, op Operator, addonID int64, req dto.OrderAddonReq) (*model.OrderAddon, error) {
	a, err := s.AddonRepo.GetByID(ctx, op.CompanyID, addonID)
	if err != nil {
		return nil, errs.NotFound("加项不存在")
	}
	o, err := s.ensureOrderEditable(ctx, op.CompanyID, a.OrderID)
	if err != nil {
		return nil, err
	}
	qty := req.Qty
	if qty <= 0 {
		qty = 1
	}
	amount := domain.Round2(req.Price * float64(qty))
	if err := repository.Tx(func(tx *gorm.DB) error {
		if err := s.AddonRepo.WithTx(tx).Update(ctx, op.CompanyID, addonID, map[string]interface{}{
			"name":       req.Name,
			"category":   req.Category,
			"price":      req.Price,
			"qty":        qty,
			"amount":     amount,
			"confirmed":  req.Confirmed,
			"remark":     req.Remark,
			"updated_by": op.UserID,
		}); err != nil {
			return err
		}
		return s.recalcOrderAmountsTx(ctx, tx, o, "修改加项 "+req.Name, op)
	}); err != nil {
		return nil, err
	}
	a.Name, a.Category = req.Name, req.Category
	a.Price, a.Qty, a.Amount = req.Price, qty, amount
	a.Confirmed, a.Remark = req.Confirmed, req.Remark
	return a, nil
}

// DeleteOrderAddon 删除加项：软删 + 重算订单金额（同事务）
func (s *Service) DeleteOrderAddon(ctx context.Context, op Operator, addonID int64) error {
	a, err := s.AddonRepo.GetByID(ctx, op.CompanyID, addonID)
	if err != nil {
		return errs.NotFound("加项不存在")
	}
	o, err := s.ensureOrderEditable(ctx, op.CompanyID, a.OrderID)
	if err != nil {
		return err
	}
	return repository.Tx(func(tx *gorm.DB) error {
		if err := s.AddonRepo.WithTx(tx).Delete(ctx, op.CompanyID, addonID); err != nil {
			return err
		}
		return s.recalcOrderAmountsTx(ctx, tx, o, "删除加项 "+a.Name, op)
	})
}

// ensureOrderEditable 取订单并校验可编辑（终态订单不允许再动金额）
func (s *Service) ensureOrderEditable(ctx context.Context, companyID, orderID int64) (*model.Order, error) {
	o, err := s.OrderRepo.GetByID(ctx, companyID, orderID)
	if err != nil {
		return nil, errs.NotFound(errs.ErrOrderNotFound)
	}
	if o.Status == enum.OrderStatusCompleted || o.Status == enum.OrderStatusCancelled {
		return nil, errs.BadRequest("订单已完成或已取消，不可修改加项")
	}
	return o, nil
}

// recalcOrderAmountsTx 按当前加项合计重算订单金额（必须在事务内调用）。
// 口径与下单一致：total = 基础套餐价 + 加项合计；deposit 由套餐定金比例推导；final = total - deposit。
func (s *Service) recalcOrderAmountsTx(ctx context.Context, tx *gorm.DB, o *model.Order, action string, op Operator) error {
	addons, err := s.AddonRepo.WithTx(tx).ListByOrder(ctx, o.CompanyID, o.ID)
	if err != nil {
		return err
	}
	var addonSum float64
	for _, a := range addons {
		addonSum += a.Amount
	}
	addonSum = domain.Round2(addonSum)

	depositRate := 0.0
	if pkg, err := s.PackageRepo.WithTx(tx).GetByID(ctx, o.CompanyID, o.PackageID); err == nil {
		depositRate = pkg.DepositRate
	}
	deposit, final, total := domain.SplitOrderAmounts(o.BasePrice, depositRate, addonSum)

	if err := s.OrderRepo.WithTx(tx).Update(ctx, o.CompanyID, o.ID, map[string]interface{}{
		"addon_amount": addonSum,
		"deposit_amt":  deposit,
		"final_amt":    final,
		"total_amt":    total,
		"updated_by":   op.UserID,
	}); err != nil {
		return err
	}
	return s.writeOrderLogTx(ctx, tx, o.ID, "addon_change", o.Status, o.Status,
		fmt.Sprintf("%s：加项合计 ¥%.2f，订单总额 ¥%.2f", action, addonSum, total), op)
}
