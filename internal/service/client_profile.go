package service

import (
	"context"
	"strings"
	"time"

	"photography-server/internal/contract"
	"photography-server/internal/pkg/errs"
	"photography-server/internal/repository"
)

// client_profile 客户中心（H5 CC01）—— 个人资料读写 + 我的评价。
//
// 归属铁律：一律以令牌内的 cu.CustomerID 为准，**不接受**请求体传 customer_id，
// 否则客户能改/看别人的档案（#29 同款防遍历要求）。
// 可改字段白名单见 contract.ClientProfileUpdateReq —— crm_customer 是与员工端共用的表，
// 内部字段（remark / tags / level / source / status）不在白名单内。

// 性别取值（biz_crm_customer.gender 列注释：male-男 female-女 unknown-未知）。
// 后端此前无该校验，客户端自助提交必须收口，否则自由文本会污染 PC 端筛选与展示。
var customerGenders = map[string]bool{"": true, "male": true, "female": true, "unknown": true}

// ClientProfile 我的资料
func (s *Service) ClientProfile(ctx context.Context, cu *ClientUser) (*contract.ClientProfileResp, error) {
	c, err := s.CustomerRepo.GetByID(ctx, cu.CompanyID, cu.CustomerID)
	if err != nil {
		return nil, errs.NotFound(errs.ErrCustomerNotFound)
	}
	return contract.NewClientProfileResp(c), nil
}

// ClientUpdateProfile 客户自助修改资料。
// 每个字段「指针非 nil 才写」：nil = 未提交，显式空串 = 清空（清偏好必须能生效）。
func (s *Service) ClientUpdateProfile(ctx context.Context, cu *ClientUser, req contract.ClientProfileUpdateReq) (*contract.ClientProfileResp, error) {
	updates := map[string]interface{}{}

	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return nil, errs.BadRequest(errs.ErrCustomerNameRequired) // 订单/评价的客户姓名会跟着变，空姓名不可读
		}
		if len([]rune(name)) > 50 { // 列宽 varchar(50)
			return nil, errs.BadRequest(errs.ErrCustomerNameTooLong)
		}
		updates["name"] = name
	}
	if req.Wechat != nil {
		wechat := strings.TrimSpace(*req.Wechat)
		if len([]rune(wechat)) > 50 {
			return nil, errs.BadRequest(errs.ErrCustomerWechatTooLong)
		}
		updates["wechat"] = wechat
	}
	if req.Gender != nil {
		gender := strings.TrimSpace(*req.Gender)
		if !customerGenders[gender] {
			return nil, errs.BadRequest(errs.ErrCustomerGenderInvalid)
		}
		updates["gender"] = gender
	}
	if req.Birthday != nil {
		birthday := strings.TrimSpace(*req.Birthday)
		if birthday != "" {
			// 只收 2006-01-02：生日列是字符串，放开格式会让「2026/1/2」「2026年1月2日」
			// 混进同一列，PC 端排序与年龄计算全部失效。
			if _, err := time.Parse("2006-01-02", birthday); err != nil {
				return nil, errs.BadRequest(errs.ErrCustomerBirthdayFormatInvalid)
			}
		}
		updates["birthday"] = birthday
	}
	if req.PreferStyle != nil {
		style := strings.TrimSpace(*req.PreferStyle)
		if len([]rune(style)) > 100 { // 列宽 varchar(100)
			return nil, errs.BadRequest(errs.ErrCustomerPreferStyleTooLong)
		}
		updates["prefer_style"] = style
	}
	if req.PreferScene != nil {
		scene := strings.TrimSpace(*req.PreferScene)
		if len([]rune(scene)) > 100 {
			return nil, errs.BadRequest(errs.ErrCustomerPreferSceneTooLong)
		}
		updates["prefer_scene"] = scene
	}

	// 一个字段都没提交：幂等返回当前资料（客户端「保存」空表单不应报错）
	if len(updates) == 0 {
		return s.ClientProfile(ctx, cu)
	}
	if err := s.CustomerRepo.Update(ctx, cu.CompanyID, cu.CustomerID, updates); err != nil {
		return nil, err
	}
	// 姓名会进客户认证画像缓存（middleware.loadCustomerProfile → authcache.CustomerProfile），
	// 改完必须失效：否则客户中心已显示新名字，其它接口在缓存 TTL 内仍用旧名字。
	s.invalidateCustomerCache(ctx, cu.CompanyID, cu.CustomerID)
	return s.ClientProfile(ctx, cu)
}

// ClientReviews 我的评价（客户中心 → 我的评价，只读）。
// 归属由 customer_id 锁定；评价连带订单快照（编号/套餐/拍摄日期），客户无需逐单回查。
func (s *Service) ClientReviews(ctx context.Context, cu *ClientUser) ([]repository.ReviewListItem, error) {
	return s.ReviewRepo.ListByCustomer(ctx, cu.CompanyID, cu.CustomerID)
}
