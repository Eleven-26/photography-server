package controller

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"photography-server/internal/middleware"
	"photography-server/internal/presentation/dto"
	"photography-server/internal/response"
)

func (h *Controller) LeadList(c *gin.Context) {
	op := middleware.GetOperator(c)
	page, pageSize := pager(c)
	ownerID, _ := strconv.ParseInt(c.Query("owner_id"), 10, 64)
	list, total, err := h.Svc.ListLeads(c.Request.Context(), op, page, pageSize,
		queryStr(c, "keyword"), queryStr(c, "status"), ownerID)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.PageOK(c, list, total, page, pageSize)
}

func (h *Controller) LeadDetail(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := pathID(c)
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
	if err := h.bindJSON(c, &req); err != nil {
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
	id, err := pathID(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req dto.LeadUpdateReq
	if err := h.bindJSON(c, &req); err != nil {
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
	id, err := pathID(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req dto.LeadFollowReq
	if err := h.bindJSON(c, &req); err != nil {
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
	id, err := pathID(c)
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
	id, err := pathParam(c, "lead_id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req dto.QuoteCreateReq
	if err := h.bindJSON(c, &req); err != nil {
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
	response.OK(c, []interface{}{})
}

func (h *Controller) QuoteStatus(c *gin.Context) {
	response.OKNil(c)
}
