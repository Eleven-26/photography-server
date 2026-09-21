// 站内通知（由 h5.go 按业务域拆分，2026-09-21 结构整改）。
package h5

import (
	"photography-server/internal/middleware"
	"photography-server/internal/pkg/params"
	"photography-server/internal/pkg/response"
	"photography-server/internal/presentation/bind"

	"github.com/gin-gonic/gin"
)

// NotificationList 我的通知列表（body: unread=1 只看未读）
// @Summary      我的通知列表
// @Description  客户本人的站内通知（分页）；unread=1 只看未读。
// @Tags         客户区·通知
// @Accept       json
// @Produce      json
// @Param        req  body  object{page=int,page_size=int,unread=string}  true  "查询条件（unread=1 只看未读）"
// @Success      200  {object}  response.Body{data=response.Page{list=[]model.SysNotification,total=int,page=int,page_size=int}}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /h5/notification/list [post]
func (h *Controller) NotificationList(c *gin.Context) {
	cu := middleware.GetClientUser(c)
	page, pageSize := bind.Pager(c)
	list, total, err := h.Svc.ListClientNotifications(c.Request.Context(), cu, page, pageSize, params.Str(c, "unread") == "1")
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.PageOK(c, list, total, page, pageSize)
}

// NotificationUnreadCount 未读通知数（铃铛红点）
// @Summary      未读通知数
// @Description  铃铛红点用。
// @Tags         客户区·通知
// @Accept       json
// @Produce      json
// @Success      200  {object}  response.Body{data=object{count=int}}
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /h5/notification/unread-count [post]
func (h *Controller) NotificationUnreadCount(c *gin.Context) {
	cu := middleware.GetClientUser(c)
	count, err := h.Svc.UnreadClientNotificationCount(c.Request.Context(), cu)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{"count": count})
}

// NotificationRead 标记单条已读
// @Summary      标记通知已读
// @Tags         客户区·通知
// @Accept       json
// @Produce      json
// @Param        id  path  int  true  "通知ID"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /h5/notification/read/{id} [post]
func (h *Controller) NotificationRead(c *gin.Context) {
	cu := middleware.GetClientUser(c)
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.MarkClientNotificationRead(c.Request.Context(), cu, id); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

// NotificationReadAll 全部标记已读
// @Summary      全部标记已读
// @Tags         客户区·通知
// @Accept       json
// @Produce      json
// @Success      200  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /h5/notification/read-all [post]
func (h *Controller) NotificationReadAll(c *gin.Context) {
	cu := middleware.GetClientUser(c)
	if err := h.Svc.MarkAllClientNotificationsRead(c.Request.Context(), cu); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}
