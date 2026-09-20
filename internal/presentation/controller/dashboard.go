package controller

import (
	"github.com/gin-gonic/gin"

	"photography-server/internal/middleware"
	"photography-server/internal/pkg/response"
	"photography-server/internal/presentation/bind"
)

// @Summary      数据看板
// @Description  返回首页经营概览（订单/客户/收款等聚合指标与今日拍摄、待办）。
// @Tags         数据看板
// @Accept       json
// @Produce      json
// @Success      200  {object}  response.Body{data=contract.DashboardOverviewResp}
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /dashboard/overview [post]
func (h *Controller) DashboardOverview(c *gin.Context) {
	op := middleware.GetOperator(c)
	ov, err := h.Svc.Overview(c.Request.Context(), op)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, ov)
}

// @Summary      通知列表
// @Description  分页查询当前操作人的站内通知（员工端按 receiver_type=1 + 操作人隔离）。
// @Tags         通知
// @Accept       json
// @Produce      json
// @Param        req  body  object{page=int,page_size=int,unread=string}  true  "查询条件（unread=1 只看未读）"
// @Success      200  {object}  response.Body{data=response.Page{list=[]model.SysNotification,total=int,page=int,page_size=int}}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /notification/list [post]
func (h *Controller) NotificationList(c *gin.Context) {
	op := middleware.GetOperator(c)
	page, pageSize := bind.Pager(c)
	list, total, err := h.Svc.ListNotifications(c.Request.Context(), op, page, pageSize, bind.ParamStr(c, "unread") == "1")
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.PageOK(c, list, total, page, pageSize)
}

// @Summary      未读通知数
// @Description  返回当前操作人的未读通知数量（用于角标）。
// @Tags         通知
// @Accept       json
// @Produce      json
// @Success      200  {object}  response.Body{data=object{unread=int}}
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /notification/unread-count [post]
func (h *Controller) NotificationUnreadCount(c *gin.Context) {
	op := middleware.GetOperator(c)
	count, err := h.Svc.UnreadNotificationCount(c.Request.Context(), op)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, map[string]int64{"unread": count})
}

// @Summary      标记通知已读
// @Description  把指定通知标记为已读（仅能操作本人的通知）。
// @Tags         通知
// @Accept       json
// @Produce      json
// @Param        id  path  int  true  "通知ID"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /notification/read/{id} [post]
func (h *Controller) NotificationRead(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := bind.PathID(c, "id")
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

// @Summary      全部标记已读
// @Description  把当前操作人的全部未读通知标记为已读。
// @Tags         通知
// @Accept       json
// @Produce      json
// @Success      200  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /notification/read-all [post]
func (h *Controller) NotificationReadAll(c *gin.Context) {
	op := middleware.GetOperator(c)
	if err := h.Svc.MarkAllNotificationsRead(c.Request.Context(), op); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}
