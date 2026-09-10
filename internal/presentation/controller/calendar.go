package controller

import (
	"github.com/gin-gonic/gin"

	"photography-server/internal/middleware"
	"photography-server/internal/pkg/params"
	"photography-server/internal/presentation/dto"
	"photography-server/internal/response"
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
