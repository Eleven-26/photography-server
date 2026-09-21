// 线索 AI 简报（由 staff.go 按业务域拆分，2026-09-21 结构整改）。
package wechat

import (
	"photography-server/internal/contract"
	"photography-server/internal/middleware"
	"photography-server/internal/pkg/response"
	"photography-server/internal/presentation/bind"

	"github.com/gin-gonic/gin"
)

// BriefGenerate 生成线索 AI 简报
// @Summary      生成线索 AI 简报
// @Description  依据线索信息重建需求摘要（已确认项 + 待追问项），**覆盖旧数据**。
// @Tags         员工端·线索简报
// @Accept       json
// @Produce      json
// @Param        lead_id  path  int  true  "线索ID"
// @Success      200  {object}  response.Body{data=[]model.LeadBriefItem}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /wechat/staff/brief/generate/{lead_id} [post]
func (h *Controller) BriefGenerate(c *gin.Context) {
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

// BriefList 简报项列表
// @Summary      简报项列表
// @Tags         员工端·线索简报
// @Accept       json
// @Produce      json
// @Param        lead_id  path  int  true  "线索ID"
// @Success      200  {object}  response.Body{data=[]model.LeadBriefItem}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /wechat/staff/brief/list/{lead_id} [post]
func (h *Controller) BriefList(c *gin.Context) {
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

// BriefSend 发送追问
// @Summary      发送追问
// @Description  把简报项作为追问消息发送给客户。
// @Tags         员工端·线索简报
// @Accept       json
// @Produce      json
// @Param        id  path  int  true  "简报项ID"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /wechat/staff/brief/send/{id} [post]
func (h *Controller) BriefSend(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.StaffBriefSend(c.Request.Context(), op, id); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

// BriefConfirm 确认简报项
// @Summary      确认简报项
// @Description  人工确认/修正 AI 提取出的简报项取值（AI 提取结果需人工复核后生效）。
// @Tags         员工端·线索简报
// @Accept       json
// @Produce      json
// @Param        id   path  int                             true  "简报项ID"
// @Param        req  body  contract.StaffBriefConfirmReq    true  "确认值"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /wechat/staff/brief/confirm/{id} [post]
func (h *Controller) BriefConfirm(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req contract.StaffBriefConfirmReq
	if err := bind.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.StaffBriefConfirm(c.Request.Context(), op, id, req.Value); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}
