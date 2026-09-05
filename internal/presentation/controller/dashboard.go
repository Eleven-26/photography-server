package controller

import (
	"github.com/gin-gonic/gin"

	"photography-server/internal/middleware"
	"photography-server/internal/response"
)

func (h *Controller) DashboardOverview(c *gin.Context) {
	op := middleware.GetOperator(c)
	ov, err := h.Svc.Overview(c.Request.Context(), op)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, ov)
}

func (h *Controller) NotificationList(c *gin.Context) {
	op := middleware.GetOperator(c)
	page, pageSize := pager(c)
	list, total, err := h.Svc.ListNotifications(c.Request.Context(), op, page, pageSize, queryStr(c, "unread") == "1")
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.PageOK(c, list, total, page, pageSize)
}

func (h *Controller) NotificationUnreadCount(c *gin.Context) {
	op := middleware.GetOperator(c)
	count, err := h.Svc.UnreadNotificationCount(c.Request.Context(), op)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, map[string]int64{"unread": count})
}

func (h *Controller) NotificationRead(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := pathID(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.MarkNotificationRead(c.Request.Context(), op, id); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

func (h *Controller) NotificationReadAll(c *gin.Context) {
	op := middleware.GetOperator(c)
	if err := h.Svc.MarkAllNotificationsRead(c.Request.Context(), op); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}
