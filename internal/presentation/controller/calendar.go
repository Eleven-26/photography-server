package controller

import (
	"github.com/gin-gonic/gin"

	"photography-server/internal/middleware"
	"photography-server/internal/response"
	"photography-server/internal/service"
)

func (h *Controller) CalendarList(c *gin.Context) {
	op := middleware.GetOperator(c)
	list, err := h.Svc.ListCalendar(op, queryStr(c, "start_date"), queryStr(c, "end_date"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, list)
}

func (h *Controller) CalendarLock(c *gin.Context) {
	op := middleware.GetOperator(c)
	var req service.BlockReq
	if err := h.bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.LockBlock(op, req); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

func (h *Controller) CalendarCancel(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := pathID(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.CancelBlock(op, id); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}
