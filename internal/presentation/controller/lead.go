package controller

import (
	"github.com/gin-gonic/gin"

	"photography-server/internal/middleware"
	"photography-server/internal/pkg/params"
	"photography-server/internal/presentation/bind"
	"photography-server/internal/presentation/dto"
	"photography-server/internal/presentation/response"
)

func (h *Controller) LeadList(c *gin.Context) {
	op := middleware.GetOperator(c)
	page, pageSize := bind.Pager(c)
	ownerID := params.Int64(c, "owner_id")
	list, total, err := h.Svc.ListLeads(c.Request.Context(), op, page, pageSize,
		bind.ParamStr(c, "keyword"), bind.ParamStr(c, "status"), ownerID)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.PageOK(c, list, total, page, pageSize)
}

func (h *Controller) LeadDetail(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	lead, err := h.Svc.GetLeadDetail(c.Request.Context(), op, id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, lead)
}

func (h *Controller) LeadCreate(c *gin.Context) {
	op := middleware.GetOperator(c)
	var req dto.LeadCreateReq
	if err := bind.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	lead, err := h.Svc.CreateLead(c.Request.Context(), op, req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, lead)
}

func (h *Controller) LeadUpdate(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req dto.LeadUpdateReq
	if err := bind.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.UpdateLead(c.Request.Context(), op, id, req); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

func (h *Controller) LeadDelete(c *gin.Context) {
	response.OKNil(c)
}

func (h *Controller) LeadFollow(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req dto.LeadFollowReq
	if err := bind.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.FollowLead(c.Request.Context(), op, id, req); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

func (h *Controller) LeadConvert(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	customer, err := h.Svc.ConvertLeadToCustomer(c.Request.Context(), op, id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, customer)
}

// -------- 报价单 --------

func (h *Controller) QuoteCreate(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := bind.PathID(c, "lead_id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req dto.QuoteCreateReq
	if err := bind.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	quote, err := h.Svc.CreateQuote(c.Request.Context(), op, id, req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, quote)
}

func (h *Controller) QuoteList(c *gin.Context) {
	op := middleware.GetOperator(c)
	leadID, err := bind.PathID(c, "lead_id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	list, err := h.Svc.ListQuotes(c.Request.Context(), op, leadID)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, list)
}

func (h *Controller) QuoteStatus(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req dto.QuoteStatusReq
	if err := bind.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.UpdateQuoteStatus(c.Request.Context(), op, id, req.Status); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

// -------- 沟通记录 / 需求摘要（复用员工端已实现的线索扩展能力） --------

// LeadMessages 线索沟通记录（客户来讯 + 工作室发出）
func (h *Controller) LeadMessages(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	list, err := h.Svc.StaffLeadMessages(c.Request.Context(), op, id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, list)
}

// LeadMessageSend 发送沟通消息（追问/报价通知/作品分享）
func (h *Controller) LeadMessageSend(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req dto.StaffLeadMessageReq
	if err := bind.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	m, err := h.Svc.StaffSendLeadMessage(c.Request.Context(), op, id, req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, m)
}

// LeadBriefList 需求摘要项列表（已确认 + 待追问）
func (h *Controller) LeadBriefList(c *gin.Context) {
	op := middleware.GetOperator(c)
	leadID, err := bind.PathID(c, "lead_id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	items, err := h.Svc.StaffBriefList(c.Request.Context(), op, leadID)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, items)
}

// LeadBriefGenerate 依据线索信息重建需求摘要（已确认项 + 待追问项，覆盖旧数据）
func (h *Controller) LeadBriefGenerate(c *gin.Context) {
	op := middleware.GetOperator(c)
	leadID, err := bind.PathID(c, "lead_id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	items, err := h.Svc.StaffBriefGenerate(c.Request.Context(), op, leadID)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, items)
}
