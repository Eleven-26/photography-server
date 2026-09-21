// 交付反馈（由 staff.go 按业务域拆分，2026-09-21 结构整改）。
package wechat

import (
	"photography-server/internal/contract"
	"photography-server/internal/middleware"
	"photography-server/internal/pkg/params"
	"photography-server/internal/pkg/response"
	"photography-server/internal/presentation/bind"

	"github.com/gin-gonic/gin"
)

// FeedbackList 客户修图反馈列表（body: status 1-待处理 2-已处理，0-全部）
// @Summary      修图反馈列表
// @Description  客户修图反馈的待办列表（分页）。
// @Tags         员工端·反馈整理
// @Accept       json
// @Produce      json
// @Param        req  body  object{page=int,page_size=int,status=int}  true  "查询条件（status 1-待处理 2-已处理，0=全部）"
// @Success      200  {object}  response.Body{data=response.Page{list=[]repository.FeedbackListItem,total=int,page=int,page_size=int}}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /wechat/staff/delivery/feedback/list [post]
func (h *Controller) FeedbackList(c *gin.Context) {
	op := middleware.GetOperator(c)
	page, pageSize := bind.Pager(c)
	list, total, err := h.Svc.ListFeedbackItems(c.Request.Context(), op, params.Int(c, "status"), page, pageSize)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.PageOK(c, list, total, page, pageSize)
}

// FeedbackHandle 标记反馈已处理并记录处理备注
// @Summary      处理反馈
// @Description  标记反馈已处理并记录处理备注。⚠️ :item_id 是**交付文件 ID**。
// @Tags         员工端·反馈整理
// @Accept       json
// @Produce      json
// @Param        item_id  path  int                              true  "交付文件ID"
// @Param        req      body  contract.StaffFeedbackHandleReq   true  "处理备注"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /wechat/staff/delivery/feedback/handle/{item_id} [post]
func (h *Controller) FeedbackHandle(c *gin.Context) {
	op := middleware.GetOperator(c)
	itemID, err := bind.PathID(c, "item_id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req contract.StaffFeedbackHandleReq
	if err := bind.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.HandleFeedbackItem(c.Request.Context(), op, itemID, req.Remark); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

// FeedbackSubmit 提交意见反馈（body: contract.FeedbackSubmitReq）
// @Summary      提交意见反馈
// @Description  员工提交意见反馈，流转在管理后台处理。免权限点（提交人本人）。
// @Tags         员工端·个人中心
// @Accept       json
// @Produce      json
// @Param        req  body  contract.FeedbackSubmitReq  true  "反馈内容"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /wechat/staff/feedback/submit [post]
func (h *Controller) FeedbackSubmit(c *gin.Context) {
	op := middleware.GetOperator(c)
	var req contract.FeedbackSubmitReq
	if err := bind.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.SubmitFeedback(c.Request.Context(), op, req); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}
