// 评价（由 h5.go 按业务域拆分，2026-09-21 结构整改）。
package h5

import (
	"photography-server/internal/contract"
	"photography-server/internal/middleware"
	"photography-server/internal/pkg/response"
	"photography-server/internal/presentation/bind"

	"github.com/gin-gonic/gin"
)

// ReviewCreate 评价订单
// @Summary      评价订单
// @Tags         客户区·评价
// @Accept       json
// @Produce      json
// @Param        order_id  path  int                     true  "订单ID"
// @Param        req       body  contract.ClientReviewReq true  "评价内容"
// @Success      200  {object}  response.Body{data=model.OrderReview}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /h5/review/create/{order_id} [post]
func (h *Controller) ReviewCreate(c *gin.Context) {
	cu := middleware.GetClientUser(c)
	orderID, err := bind.PathID(c, "order_id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req contract.ClientReviewReq
	if err := bind.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	rv, err := h.Svc.ClientReviewCreate(c.Request.Context(), cu, orderID, req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, rv)
}

// ReviewList 我的评价（客户中心 → 我的评价，只读；不分页的业务集合，同改期/退款列表口径）
// @Summary      我的评价
// @Description  客户中心 → 我的评价，只读；按令牌内 customer_id 锁定归属（**不分页**，同改期/退款列表口径）。
// @Tags         客户区·评价
// @Accept       json
// @Produce      json
// @Success      200  {object}  response.Body{data=[]repository.ReviewListItem}
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /h5/review/list [post]
func (h *Controller) ReviewList(c *gin.Context) {
	cu := middleware.GetClientUser(c)
	list, err := h.Svc.ClientReviews(c.Request.Context(), cu)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, list)
}
