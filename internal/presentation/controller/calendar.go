package controller

import (
	"github.com/gin-gonic/gin"

	"photography-server/internal/middleware"
	"photography-server/internal/pkg/params"
	"photography-server/internal/presentation/dto"
	"photography-server/internal/presentation/response"
)

func (h *Controller) CalendarList(c *gin.Context) {
	op := middleware.GetOperator(c)
	photographerID := params.Int64(c, "photographer_id")
	list, err := h.Svc.ListCalendar(c.Request.Context(), op, queryStr(c, "start_date"), queryStr(c, "end_date"), photographerID)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, list)
}

func (h *Controller) CalendarLock(c *gin.Context) {
	op := middleware.GetOperator(c)
	var req dto.CalendarBlockReq
	if err := h.bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	block, err := h.Svc.BlockCalendar(c.Request.Context(), op, req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, block)
}

func (h *Controller) CalendarCancel(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := pathID(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.CancelCalendarBlock(c.Request.Context(), op, id); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

// ---------------------------------------------------------------------
// 档期规则（排班时段模板）—— 与员工端共用同一 service，PC 端仅暴露入口
// ---------------------------------------------------------------------

// SlotTemplateList 档期时段模板列表
func (h *Controller) SlotTemplateList(c *gin.Context) {
	op := middleware.GetOperator(c)
	list, err := h.Svc.SlotTemplates(c.Request.Context(), op, params.Int64(c, "photographer_id"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, list)
}

// SlotTemplateSave 新建（无 :id）/ 更新（有 :id）档期时段模板
func (h *Controller) SlotTemplateSave(c *gin.Context) {
	op := middleware.GetOperator(c)
	var id int64
	if raw := c.Param("id"); raw != "" {
		parsed, err := pathID(c)
		if err != nil {
			response.Fail(c, err)
			return
		}
		id = parsed
	}
	var req dto.StaffSlotTemplateReq
	if err := h.bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	m, err := h.Svc.SaveSlotTemplate(c.Request.Context(), op, id, req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, m)
}

// SlotTemplateDelete 删除档期时段模板
func (h *Controller) SlotTemplateDelete(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := pathID(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.DeleteSlotTemplate(c.Request.Context(), op, id); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}
