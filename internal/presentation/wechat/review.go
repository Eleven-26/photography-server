// 评价（由 staff.go 按业务域拆分，2026-09-21 结构整改）。
package wechat

import (
	"photography-server/internal/contract"
	"photography-server/internal/middleware"
	"photography-server/internal/pkg/params"
	"photography-server/internal/pkg/response"
	"photography-server/internal/presentation/bind"

	"github.com/gin-gonic/gin"
)

// ReviewList 评价列表
// @Summary      评价列表
// @Description  分页查询客户评价，可按最低星级过滤。
// @Tags         员工端·评价
// @Accept       json
// @Produce      json
// @Param        req  body  object{page=int,page_size=int,min_rating=int}  true  "查询条件（min_rating 0=不限）"
// @Success      200  {object}  response.Body{data=response.Page{list=[]model.OrderReview,total=int,page=int,page_size=int}}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /wechat/staff/review/list [post]
func (h *Controller) ReviewList(c *gin.Context) {
	op := middleware.GetOperator(c)
	page, pageSize := bind.Pager(c)
	minRating := params.Int(c, "min_rating")
	list, total, err := h.Svc.StaffReviewList(c.Request.Context(), op, page, pageSize, minRating)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.PageOK(c, list, total, page, pageSize)
}

// ReviewReply 回复评价
// @Summary      回复评价
// @Tags         员工端·评价
// @Accept       json
// @Produce      json
// @Param        id   path  int                            true  "评价ID"
// @Param        req  body  contract.StaffReviewReplyReq    true  "回复内容"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /wechat/staff/review/reply/{id} [post]
func (h *Controller) ReviewReply(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req contract.StaffReviewReplyReq
	if err := bind.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.StaffReviewReply(c.Request.Context(), op, id, req.Reply); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}
