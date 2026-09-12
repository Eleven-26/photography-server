package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"photography-server/internal/enum"
	"photography-server/internal/model"
	"photography-server/internal/pkg/errs"
	"photography-server/internal/presentation/dto"
)

// staff 小程序员工端能力：工作台待办、改期审批、线索沟通与 AI 简报、
// 档期时段模板、评价回复、工作室设置。复用 Operator 上下文（员工登录）。
// 订单状态流转/收款/交付上传等操作直接复用 PC 端既有 service 方法。

// StaffOverview 工作台待办统计（按状态计数，全部走现有 List 的 total）
func (s *Service) StaffOverview(ctx context.Context, op Operator) (*dto.StaffOverview, error) {
	ov := &dto.StaffOverview{}
	type countJob struct {
		run  func() (int64, error)
		dest *int64
	}
	jobs := []countJob{
		{func() (int64, error) { _, t, e := s.OrderRepo.List(ctx, op.CompanyID, 1, 1, "0", 0); return t, e }, &ov.PendingConfirm},
		{func() (int64, error) { _, t, e := s.OrderRepo.List(ctx, op.CompanyID, 1, 1, "1", 0); return t, e }, &ov.PendingDeposit},
		{func() (int64, error) { _, t, e := s.OrderRepo.List(ctx, op.CompanyID, 1, 1, "2", 0); return t, e }, &ov.PendingShoot},
		{func() (int64, error) { _, t, e := s.OrderRepo.List(ctx, op.CompanyID, 1, 1, "5", 0); return t, e }, &ov.PendingDelivery},
		{func() (int64, error) {
			_, t, e := s.RescheduleRepo.List(ctx, op.CompanyID, 1, 1, int(enum.RescheduleStatusPending))
			return t, e
		}, &ov.PendingReschedule},
		{func() (int64, error) {
			_, t, e := s.CustomRequestRepo.List(ctx, op.CompanyID, 1, 1, int(enum.CustomRequestPending), 0)
			return t, e
		}, &ov.PendingCustomRequest},
	}
	for _, j := range jobs {
		n, err := j.run()
		if err != nil {
			return nil, err
		}
		*j.dest = n
	}

	// 今日档期（未取消的档期锁即今日拍摄/待拍摄）
	today := time.Now().Format("2006-01-02")
	blocks, err := s.CalendarRepo.List(ctx, op.CompanyID, today, today, 0)
	if err != nil {
		return nil, err
	}
	for _, b := range blocks {
		if b.Status != enum.BlockStatusCancelled {
			ov.TodayShoot++
		}
	}

	// 待审批退款：只统计"申请中"的单据
	// （旧实现调用 ListRefunds 未带状态，当时仓储硬编码 status=已退款，统计口径错误）
	_, refundTotal, err := s.FinanceRepo.ListRefunds(ctx, op.CompanyID, 1, 1, fmt.Sprintf("%d", int(enum.RefundStatusApplying)))
	if err != nil {
		return nil, err
	}
	ov.PendingRefund = refundTotal
	return ov, nil
}

// ---------------------------------------------------------------------
// 改期审批
// ---------------------------------------------------------------------

// StaffRescheduleList 改期单列表（status>0 时按状态过滤）
func (s *Service) StaffRescheduleList(ctx context.Context, op Operator, page, pageSize, status int) ([]model.OrderReschedule, int64, error) {
	return s.RescheduleRepo.List(ctx, op.CompanyID, page, pageSize, status)
}

// StaffRescheduleAudit 改期审批。同意后同步更新订单拍摄日期/时间并重建档期锁；
// 拒绝仅记录审批备注。
func (s *Service) StaffRescheduleAudit(ctx context.Context, op Operator, rescheduleID int64, approved bool, remark string) error {
	rs, err := s.RescheduleRepo.GetByID(ctx, op.CompanyID, rescheduleID)
	if err != nil {
		return errs.NotFound("改期单不存在")
	}
	if rs.Status != enum.RescheduleStatusPending {
		return errs.BadRequest("该改期单已处理")
	}
	o, err := s.OrderRepo.GetByID(ctx, op.CompanyID, rs.OrderID)
	if err != nil {
		return errs.NotFound(errs.ErrOrderNotFound)
	}

	now := time.Now().Format("2006-01-02 15:04:05")
	status := enum.RescheduleStatusRejected
	if approved {
		status = enum.RescheduleStatusApproved
	}
	if err := s.RescheduleRepo.Update(ctx, op.CompanyID, rescheduleID, map[string]interface{}{
		"status":       status,
		"audit_by":     op.UserID,
		"audit_name":   op.Nickname,
		"audit_at":     now,
		"audit_remark": remark,
	}); err != nil {
		return err
	}

	if !approved {
		if err := s.writeOrderLog(ctx, rs.OrderID, "reschedule_rejected", o.Status, o.Status, "改期申请被拒绝: "+remark, op); err != nil {
			return err
		}
		s.NotifyClient(ctx, op, rs.CustomerID, "order", "改期申请未通过",
			"改期申请未通过："+remark, "reschedule", rs.ID)
		return nil
	}

	// 同意：更新订单拍摄日期时间 + 取消旧档期锁 + 建新档期锁
	if err := s.OrderRepo.Update(ctx, op.CompanyID, rs.OrderID, map[string]interface{}{
		"shoot_date": rs.NewDate,
		"shoot_time": rs.NewTime,
	}); err != nil {
		return err
	}
	_ = s.OrderRepo.UpdateCalendarBlockStatus(ctx, op.CompanyID, rs.OrderID, enum.BlockStatusCancelled)
	block := model.CalendarBlock{
		TenantBase:   model.TenantBase{Base: model.Base{CreatedAt: time.Now(), UpdatedAt: time.Now(), CreatedBy: op.UserID, UpdatedBy: op.UserID}, CompanyID: op.CompanyID},
		StoreID:      o.StoreID,
		OrderID:      o.ID,
		CustomerID:   o.CustomerID,
		CustomerName: o.CustomerName,
		Date:         rs.NewDate,
		TimeRange:    rs.NewTime,
		ProjectType:  o.PackageName,
		Status:       enum.BlockStatusLocked,
	}
	if err := s.CalendarRepo.Create(ctx, &block); err != nil {
		return err
	}
	if err := s.writeOrderLog(ctx, rs.OrderID, "reschedule_approved", o.Status, o.Status,
		"改期已同意: "+rs.OriginalDate+" → "+rs.NewDate+" "+rs.NewTime, op); err != nil {
		return err
	}
	s.NotifyClient(ctx, op, rs.CustomerID, "order", "改期申请已通过",
		"拍摄时间已调整为 "+rs.NewDate+" "+rs.NewTime, "reschedule", rs.ID)
	return nil
}

// ---------------------------------------------------------------------
// 线索沟通 + AI 简报
// ---------------------------------------------------------------------

// StaffLeadMessages 线索沟通记录（按时间正序）
func (s *Service) StaffLeadMessages(ctx context.Context, op Operator, leadID int64) ([]model.LeadMessage, error) {
	if _, err := s.LeadRepo.GetByID(ctx, op.CompanyID, leadID); err != nil {
		return nil, errs.NotFound(errs.ErrLeadNotFound)
	}
	return s.LeadExtraRepo.ListMessages(ctx, op.CompanyID, leadID)
}

// StaffSendLeadMessage 工作室发出沟通消息（追问/报价通知/作品分享）
func (s *Service) StaffSendLeadMessage(ctx context.Context, op Operator, leadID int64, req dto.StaffLeadMessageReq) (*model.LeadMessage, error) {
	if strings.TrimSpace(req.Content) == "" {
		return nil, errs.BadRequest("消息内容不能为空")
	}
	if _, err := s.LeadRepo.GetByID(ctx, op.CompanyID, leadID); err != nil {
		return nil, errs.NotFound(errs.ErrLeadNotFound)
	}
	now := time.Now()
	m := model.LeadMessage{
		TenantBase: model.TenantBase{Base: model.Base{CreatedAt: now, UpdatedAt: now, CreatedBy: op.UserID, UpdatedBy: op.UserID}, CompanyID: op.CompanyID},
		LeadID:     leadID,
		Direction:  2, // 工作室发出
		Channel:    orDefault(req.Channel, "h5"),
		Content:    req.Content,
		MsgType:    int(orDefaultInt64(req.MsgType, 2)), // 默认追问
		BizID:      req.BizID,
	}
	if err := s.LeadExtraRepo.CreateMessage(ctx, &m); err != nil {
		return nil, err
	}
	_ = s.LeadRepo.Update(ctx, op.CompanyID, leadID, map[string]interface{}{
		"follower":       1,
		"last_follow_at": now.Format("2006-01-02 15:04:05"),
	})
	return &m, nil
}

// StaffBriefGenerate 生成线索 AI 简报（规则版）。
// 已确认项来自线索档案字段；缺失项按固定话术模板生成待追问项。
// TODO(P1): 接入 LLM 按沟通记录动态生成追问话术；当前为规则模板。
func (s *Service) StaffBriefGenerate(ctx context.Context, op Operator, leadID int64) ([]model.LeadBriefItem, error) {
	l, err := s.LeadRepo.GetByID(ctx, op.CompanyID, leadID)
	if err != nil {
		return nil, errs.NotFound(errs.ErrLeadNotFound)
	}

	now := time.Now()
	confirmed := func(sort int, title, value string, pricing bool) model.LeadBriefItem {
		return model.LeadBriefItem{
			TenantBase:     model.TenantBase{Base: model.Base{CreatedAt: now, UpdatedAt: now, CreatedBy: op.UserID, UpdatedBy: op.UserID}, CompanyID: op.CompanyID},
			LeadID:         leadID,
			Title:          title,
			Value:          value,
			Status:         enum.BriefItemConfirmed,
			AffectsPricing: b2i(pricing),
			Sort:           sort,
		}
	}
	pending := func(sort int, title, question, suggestion string, pricing bool) model.LeadBriefItem {
		return model.LeadBriefItem{
			TenantBase:     model.TenantBase{Base: model.Base{CreatedAt: now, UpdatedAt: now, CreatedBy: op.UserID, UpdatedBy: op.UserID}, CompanyID: op.CompanyID},
			LeadID:         leadID,
			Title:          title,
			Question:       question,
			AiSuggestion:   suggestion,
			Status:         enum.BriefItemPending,
			AffectsPricing: b2i(pricing),
			Sort:           sort,
		}
	}

	items := make([]model.LeadBriefItem, 0, 10)
	sort := 1
	// —— 已确认项 ——
	items = append(items, confirmed(sort, "联系方式", l.Name+" "+l.Mobile, false))
	sort++
	if l.ProjectType != "" {
		items = append(items, confirmed(sort, "拍摄类型", l.ProjectType, true))
		sort++
	}
	if l.BudgetMin > 0 || l.BudgetMax > 0 {
		items = append(items, confirmed(sort, "预算区间", fmt.Sprintf("%.0f-%.0f 元", l.BudgetMin, l.BudgetMax), true))
		sort++
	}
	if l.ShootDate != nil && *l.ShootDate != "" {
		items = append(items, confirmed(sort, "意向日期", *l.ShootDate, true))
		sort++
	}
	// —— 待追问项（缺失即生成） ——
	items = append(items, pending(sort, "拍摄人数", "请问这次拍摄大概几位出镜呢？",
		"您好～方便说下这次拍摄几位出镜吗？人数会影响场地和服装建议哦", true))
	sort++
	items = append(items, pending(sort, "期望风格", "您期望的拍摄风格是？",
		"我们有清新自然、复古胶片、杂志大片等风格，您更喜欢哪种感觉呢？", false))
	sort++
	if l.ProjectType == "" {
		items = append(items, pending(sort, "拍摄类型", "您想拍什么类型的照片呢？",
			"您好～想了解下您有意向的拍摄类型吗？婚纱、写真、亲子我们都很擅长", true))
		sort++
	}
	if l.BudgetMin <= 0 && l.BudgetMax <= 0 {
		items = append(items, pending(sort, "预算范围", "您的预算范围大概是？",
			"咱们套餐从基础到高定都有，告诉我个预算区间，我帮您推荐最合适的", true))
		sort++
	}
	if l.ShootDate == nil || *l.ShootDate == "" {
		items = append(items, pending(sort, "意向档期", "您期望哪天拍摄？",
			"请问您大概什么时候方便拍摄呢？我先帮您看下档期", true))
		sort++
	}
	items = append(items, pending(sort, "妆造需求", "是否需要妆造服务？",
		"需要安排专业化妆师吗？含妆造的套餐上镜效果会更好哦", false))

	// 重建：清空旧简报项后写入
	if err := s.LeadExtraRepo.DeleteBriefItemsByLead(ctx, op.CompanyID, leadID); err != nil {
		return nil, err
	}
	if err := s.LeadExtraRepo.CreateBriefItems(ctx, items); err != nil {
		return nil, err
	}
	return items, nil
}

// StaffBriefList 简报项列表
func (s *Service) StaffBriefList(ctx context.Context, op Operator, leadID int64) ([]model.LeadBriefItem, error) {
	return s.LeadExtraRepo.ListBriefItems(ctx, op.CompanyID, leadID)
}

// StaffBriefSend 发送追问（状态 pending → sent；实际触达渠道 TODO）
func (s *Service) StaffBriefSend(ctx context.Context, op Operator, itemID int64) error {
	item, err := s.LeadExtraRepo.GetBriefItem(ctx, op.CompanyID, itemID)
	if err != nil {
		return errs.NotFound("简报项不存在")
	}
	if item.Status != enum.BriefItemPending {
		return errs.BadRequest("该简报项已处理")
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	updates := map[string]interface{}{"status": enum.BriefItemSent, "sent_at": now}
	if item.AiSuggestion != "" {
		updates["question"] = item.AiSuggestion
	}
	return s.LeadExtraRepo.UpdateBriefItem(ctx, op.CompanyID, itemID, updates)
}

// StaffBriefConfirm 确认/补充简报项取值（客户回复后录入）
func (s *Service) StaffBriefConfirm(ctx context.Context, op Operator, itemID int64, value string) error {
	if _, err := s.LeadExtraRepo.GetBriefItem(ctx, op.CompanyID, itemID); err != nil {
		return errs.NotFound("简报项不存在")
	}
	if value == "" {
		return errs.BadRequest("请填写确认内容")
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	return s.LeadExtraRepo.UpdateBriefItem(ctx, op.CompanyID, itemID, map[string]interface{}{
		"status":       enum.BriefItemConfirmed,
		"value":        value,
		"confirmed_at": now,
	})
}

// ---------------------------------------------------------------------
// 档期时段模板（PC 端与员工端共用同一份实现，此处为 PC 命名入口）
// ---------------------------------------------------------------------

// SlotTemplates 档期时段模板列表
func (s *Service) SlotTemplates(ctx context.Context, op Operator, photographerID int64) ([]model.SlotTemplate, error) {
	return s.SlotTemplateRepo.List(ctx, op.CompanyID, photographerID)
}

// SaveSlotTemplate 新建/更新档期时段模板
func (s *Service) SaveSlotTemplate(ctx context.Context, op Operator, id int64, req dto.StaffSlotTemplateReq) (*model.SlotTemplate, error) {
	if req.Weekday < 0 || req.Weekday > 6 {
		return nil, errs.BadRequest("星期参数错误（0-周日 ... 6-周六）")
	}
	if req.StartTime == "" || req.EndTime == "" || req.StartTime >= req.EndTime {
		return nil, errs.BadRequest("时间段无效")
	}
	now := time.Now()
	if id > 0 {
		if _, err := s.SlotTemplateRepo.GetByID(ctx, op.CompanyID, id); err != nil {
			return nil, errs.NotFound("模板不存在")
		}
		updates := map[string]interface{}{
			"photographer_id": req.PhotographerID,
			"weekday":         req.Weekday,
			"start_time":      req.StartTime,
			"end_time":        req.EndTime,
			"status":          req.Status,
			"updated_by":      op.UserID,
		}
		if err := s.SlotTemplateRepo.Update(ctx, op.CompanyID, id, updates); err != nil {
			return nil, err
		}
		return s.SlotTemplateRepo.GetByID(ctx, op.CompanyID, id)
	}
	m := model.SlotTemplate{
		TenantBase:     model.TenantBase{Base: model.Base{CreatedAt: now, UpdatedAt: now, CreatedBy: op.UserID, UpdatedBy: op.UserID}, CompanyID: op.CompanyID},
		StoreID:        op.StoreID,
		PhotographerID: req.PhotographerID,
		Weekday:        req.Weekday,
		StartTime:      req.StartTime,
		EndTime:        req.EndTime,
		Status:         int(orDefaultInt64(req.Status, 1)),
	}
	if err := s.SlotTemplateRepo.Create(ctx, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

// DeleteSlotTemplate 删除档期时段模板
func (s *Service) DeleteSlotTemplate(ctx context.Context, op Operator, id int64) error {
	if _, err := s.SlotTemplateRepo.GetByID(ctx, op.CompanyID, id); err != nil {
		return errs.NotFound("模板不存在")
	}
	return s.SlotTemplateRepo.Delete(ctx, op.CompanyID, id)
}

// ---------------------------------------------------------------------
// 评价回复 / 工作室设置
// ---------------------------------------------------------------------

// StaffReviewList 评价列表（minRating>0 时按最低评分过滤）
func (s *Service) StaffReviewList(ctx context.Context, op Operator, page, pageSize, minRating int) ([]model.OrderReview, int64, error) {
	return s.ReviewRepo.List(ctx, op.CompanyID, page, pageSize, minRating)
}

// StaffReviewReply 摄影师回复客户评价
func (s *Service) StaffReviewReply(ctx context.Context, op Operator, reviewID int64, reply string) error {
	if strings.TrimSpace(reply) == "" {
		return errs.BadRequest("回复内容不能为空")
	}
	updates := map[string]interface{}{
		"reply":      reply,
		"reply_at":   time.Now().Format("2006-01-02 15:04:05"),
		"updated_by": op.UserID,
	}
	return s.ReviewRepo.Update(ctx, op.CompanyID, reviewID, updates)
}

// StudioSetting 工作室设置（PC / 员工端共用，不存在时自动建默认行）
func (s *Service) StudioSetting(ctx context.Context, op Operator) (*model.StudioSetting, error) {
	return s.StudioSettingRepo.GetOrCreate(ctx, op.CompanyID)
}

// UpdateStudioSetting 工作室设置更新（updates 由 controller 按「指针非 nil 才更新」组装）
func (s *Service) UpdateStudioSetting(ctx context.Context, op Operator, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}
	updates["updated_by"] = op.UserID
	return s.StudioSettingRepo.Update(ctx, op.CompanyID, updates)
}

// StaffCustomRequests 定制需求列表（待处理优先）
func (s *Service) StaffCustomRequests(ctx context.Context, op Operator, page, pageSize, status int) ([]model.CustomRequest, int64, error) {
	return s.CustomRequestRepo.List(ctx, op.CompanyID, page, pageSize, status, 0)
}

// StaffCustomRequestRespond 响应定制需求（转化线索由既有 ConvertLeadToCustomer 承接）
func (s *Service) StaffCustomRequestRespond(ctx context.Context, op Operator, id int64, response string) error {
	req, err := s.CustomRequestRepo.GetByID(ctx, op.CompanyID, id)
	if err != nil {
		return errs.NotFound("定制需求不存在")
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	updates := map[string]interface{}{
		"status":      enum.CustomRequestResponded,
		"response":    response,
		"response_by": op.UserID,
		"response_at": now,
		"updated_by":  op.UserID,
	}
	// 认领落店：公共池（store_id=0）的需求被响应后归属响应人门店，
	// 之后按常规门店口径过滤（仅本人可见自己响应的，本店可见同店的）
	if req.StoreID == 0 && op.StoreID != 0 {
		updates["store_id"] = op.StoreID
	}
	return s.CustomRequestRepo.Update(ctx, op.CompanyID, id, updates)
}

// b2i bool 转 0/1
func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

// StaffTodayFollowUp 今日待跟进线索（报告 H7：员工端 CU01 客户/线索页的「今日待跟进」）。
// 口径：下次跟进时间已到（含逾期）且尚未成交/流失，按到期时间升序——越早到期越优先。
func (s *Service) StaffTodayFollowUp(ctx context.Context, op Operator, limit int) ([]model.Lead, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	until := time.Now().Format("2006-01-02") + " 23:59:59"
	return s.LeadRepo.ListFollowUpDue(ctx, op.CompanyID, until, limit)
}
