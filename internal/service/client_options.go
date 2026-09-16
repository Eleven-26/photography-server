package service

import (
	"context"
	"sort"

	"photography-server/internal/contract"
	"photography-server/internal/repository"
)

// client_options 客户中心 —— 定制需求页的「门店 / 摄影师」候选（H5）。
//
// 为什么要有它：定制需求此前只能靠**分享链接**带入归属（URL ?staff_id=），
// 客户自己从个人中心进来时 URL 里没有摄影师，需求就无从归属。本接口给出
// 「服务过这个客户的门店与摄影师」，让客户在页面上直接选（见 contract.ClientStoreOption）。
//
// 归属铁律：一律以令牌内的 cu.CustomerID 为准（同 client_profile.go）——
// 候选只来自该客户自己的订单/需求记录，不接受客户端传入的任何过滤条件。

// ClientPhotographerOptions 客户历史服务过的门店与摄影师（H5 定制需求页两级选择候选）。
//
// shareStaffID = 分享链接带入的分享人（无链接时为 0）：即使他尚无该客户的订单/需求记录，
// 也并入候选 —— 客户正是从他的链接进来的，理应能指定他（新客户首次分享进入的常见情形）。
func (s *Service) ClientPhotographerOptions(ctx context.Context, cu *ClientUser, shareStaffID int64) ([]contract.ClientStoreOption, error) {
	fromOrders, err := s.OrderRepo.ListCustomerStaffPairs(ctx, cu.CompanyID, cu.CustomerID)
	if err != nil {
		return nil, err
	}
	fromRequests, err := s.CustomRequestRepo.ListCustomerStaffPairs(ctx, cu.CompanyID, cu.CustomerID)
	if err != nil {
		return nil, err
	}

	storeIDs := make(map[int64]bool)
	photographerIDs := make(map[int64]bool)
	for _, list := range [][]repository.CustomerStaffPair{fromOrders, fromRequests} {
		for _, p := range list {
			if p.StoreID > 0 {
				storeIDs[p.StoreID] = true
			}
			if p.PhotographerID > 0 {
				photographerIDs[p.PhotographerID] = true
			}
		}
	}
	if shareStaffID > 0 {
		photographerIDs[shareStaffID] = true
	}

	// 摄影师：换成姓名 / 头像 / 所属门店（一次批量取，避免逐个 GetByID 的 N+1）
	users, err := s.UserRepo.ListByIDs(ctx, cu.CompanyID, sortedIDSet(photographerIDs))
	if err != nil {
		return nil, err
	}
	byStore := make(map[int64][]contract.ClientPhotographerOption)
	for _, u := range users {
		// 停用员工不进入候选：他接不了单，给了客户也会被提交接口拒
		// （见 ClientSubmitCustomRequest 的 status 校验），不如一开始就不展示。
		if u.Status != 1 {
			continue
		}
		byStore[u.StoreID] = append(byStore[u.StoreID], contract.ClientPhotographerOption{
			ID:     u.ID,
			Name:   u.Nickname,
			Avatar: u.Avatar,
		})
		// 摄影师所属门店也进候选：客户可能只在这位摄影师手上拍过，门店本身未直接出现在 pairs 里
		storeIDs[u.StoreID] = true
	}

	// 门店名：取该租户全量门店再筛（门店数量级很小，不值得为 IN 再开一个仓储方法）
	stores, err := s.UserRepo.ListStores(ctx, cu.CompanyID)
	if err != nil {
		return nil, err
	}
	storeName := make(map[int64]string, len(stores))
	for _, st := range stores {
		storeName[st.ID] = st.Name
	}

	// 排序输出：门店按 ID 升序（0 = 摄影师未分配门店的占位组，排最前），摄影师按 ID 升序。
	// 顺序稳定便于前端做「默认选中第一项」，也让同一客户的两次请求结果可比对。
	out := make([]contract.ClientStoreOption, 0, len(storeIDs))
	for _, sid := range sortedIDSet(storeIDs) {
		opts := byStore[sid]
		if opts == nil {
			opts = []contract.ClientPhotographerOption{} // 该门店下没有可选摄影师时给空数组，避免下发 null
		}
		sort.Slice(opts, func(i, j int) bool { return opts[i].ID < opts[j].ID })
		out = append(out, contract.ClientStoreOption{
			StoreID:       sid,
			StoreName:     storeName[sid], // 门店已删/未分配时为 ""，前端显示为「未指定门店」
			Photographers: opts,
		})
	}
	return out, nil
}

// sortedIDSet 把 ID 集合转成升序切片（便于稳定输出与批量查询入参确定）。
func sortedIDSet(set map[int64]bool) []int64 {
	out := make([]int64, 0, len(set))
	for id := range set {
		out = append(out, id)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}
