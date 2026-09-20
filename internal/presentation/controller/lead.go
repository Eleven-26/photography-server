package controller

import (
	"github.com/gin-gonic/gin"

	"photography-server/internal/contract"
	"photography-server/internal/middleware"
	"photography-server/internal/pkg/params"
	"photography-server/internal/pkg/response"
	"photography-server/internal/presentation/bind"
)

// @Summary      线索列表
// @Description  分页查询线索，支持关键词、状态与归属人过滤。
// @Tags         线索
// @Accept       json
// @Produce      json
// @Param        req  body  object{page=int,page_size=int,keyword=string,status=string,owner_id=int}  true  "查询条件（status 见 enum.LeadStatus；owner_id 0=不过滤）"
// @Success      200  {object}  response.Body{data=response.Page{list=[]model.Lead,total=int,page=int,page_size=int}}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /lead/list [post]
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

// @Summary      线索详情
// @Description  返回线索详情（含客户来讯与需求摘要）。
// @Tags         线索
// @Accept       json
// @Produce      json
// @Param        id  path  int  true  "线索ID"
// @Success      200  {object}  response.Body{data=model.Lead}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /lead/detail/{id} [post]
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

// @Summary      新建线索
// @Description  新建线索。**有手机号时在同一事务内自动建档客户**（复用 FindOrCreateCustomerByMobile，手机号为租户内唯一键）；无手机号不建档。
// @Tags         线索
// @Accept       json
// @Produce      json
// @Param        req  body  contract.LeadCreateReq  true  "线索信息"
// @Success      200  {object}  response.Body{data=model.Lead}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /lead/create [post]
func (h *Controller) LeadCreate(c *gin.Context) {
	op := middleware.GetOperator(c)
	var req contract.LeadCreateReq
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

// @Summary      更新线索
// @Description  更新线索；body.id 为主键。
// @Tags         线索
// @Accept       json
// @Produce      json
// @Param        req  body  contract.LeadUpdateReq  true  "线索信息（含 id）"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /lead/update [post]
func (h *Controller) LeadUpdate(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := bind.BodyID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req contract.LeadUpdateReq
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

// @Summary      删除线索
// @Description  ⚠️ 当前为**占位实现**：handler 直接返回成功，未做实际删除（路由已挂 /lead/delete/:id）。
// @Tags         线索
// @Accept       json
// @Produce      json
// @Param        id  path  int  true  "线索ID"
// @Success      200  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /lead/delete/{id} [post]
func (h *Controller) LeadDelete(c *gin.Context) {
	response.OKNil(c)
}

// @Summary      记录线索跟进
// @Description  写入一条跟进记录，并刷新线索的跟进时间与归属。
// @Tags         线索
// @Accept       json
// @Produce      json
// @Param        id   path  int                     true  "线索ID"
// @Param        req  body  contract.LeadFollowReq  true  "跟进内容"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /lead/follow/{id} [post]
func (h *Controller) LeadFollow(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req contract.LeadFollowReq
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

// @Summary      线索转客户
// @Description  把线索转为客户档案（幂等：线索已有 customer_id 时复用既有客户，不重复建档）。
// @Tags         线索
// @Accept       json
// @Produce      json
// @Param        id  path  int  true  "线索ID"
// @Success      200  {object}  response.Body{data=model.Customer}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /lead/convert/{id} [post]
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

// @Summary      创建报价
// @Description  针对线索创建报价单；body.lead_id 为线索主键。
// @Tags         报价
// @Accept       json
// @Produce      json
// @Param        req  body  contract.QuoteCreateReq  true  "报价信息（含 lead_id）"
// @Success      200  {object}  response.Body{data=model.Quote}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /quote/create [post]
func (h *Controller) QuoteCreate(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := bind.BodyID(c, "lead_id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req contract.QuoteCreateReq
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

// @Summary      线索报价列表
// @Description  返回某条线索下的全部报价单（不分页）。
// @Tags         报价
// @Accept       json
// @Produce      json
// @Param        lead_id  path  int  true  "线索ID"
// @Success      200  {object}  response.Body{data=[]model.Quote}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /quote/list/{lead_id} [post]
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

// @Summary      更新报价状态
// @Description  流转报价单状态（草稿/已发送/已接受/已拒绝等，见 enum.QuoteStatus）。
// @Tags         报价
// @Accept       json
// @Produce      json
// @Param        id   path  int                       true  "报价单ID"
// @Param        req  body  contract.QuoteStatusReq   true  "目标状态"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /quote/status/{id} [post]
func (h *Controller) QuoteStatus(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req contract.QuoteStatusReq
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
// @Summary      线索沟通记录
// @Description  返回线索的双向沟通记录（客户来讯 + 工作室发出）。与员工端复用同一实现。
// @Tags         线索
// @Accept       json
// @Produce      json
// @Param        id  path  int  true  "线索ID"
// @Success      200  {object}  response.Body{data=[]model.LeadMessage}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /lead/messages/{id} [post]
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
// @Summary      发送沟通消息
// @Description  向线索发送沟通消息（追问 / 报价通知 / 作品分享）。与员工端复用同一实现。
// @Tags         线索
// @Accept       json
// @Produce      json
// @Param        id   path  int                             true  "线索ID"
// @Param        req  body  contract.StaffLeadMessageReq    true  "消息内容"
// @Success      200  {object}  response.Body{data=model.LeadMessage}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /lead/message/send/{id} [post]
func (h *Controller) LeadMessageSend(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req contract.StaffLeadMessageReq
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
// @Summary      需求摘要列表
// @Description  返回线索的需求摘要项（已确认项 + 待追问项）。与员工端 AI 简报复用同一实现。
// @Tags         线索
// @Accept       json
// @Produce      json
// @Param        lead_id  path  int  true  "线索ID"
// @Success      200  {object}  response.Body{data=[]model.LeadBriefItem}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /lead/brief/list/{lead_id} [post]
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
// @Summary      重建需求摘要
// @Description  依据线索信息重建需求摘要（已确认项 + 待追问项），**覆盖旧数据**。
// @Tags         线索
// @Accept       json
// @Produce      json
// @Param        lead_id  path  int  true  "线索ID"
// @Success      200  {object}  response.Body{data=[]model.LeadBriefItem}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /lead/brief/generate/{lead_id} [post]
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
