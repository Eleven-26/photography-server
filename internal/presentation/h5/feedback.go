// 交付反馈（由 h5.go 按业务域拆分，2026-09-21 结构整改）。
package h5

import (
	"photography-server/internal/contract"
	"photography-server/internal/middleware"
	"photography-server/internal/pkg/response"
	"photography-server/internal/presentation/bind"

	"github.com/gin-gonic/gin"
)

// FeedbackSubmit 提交精修反馈
// @Summary      提交精修反馈
// @Description  ⚠️ 本接口的 :item_id 是**交付文件 ID**。
// @Tags         客户区·交付
// @Accept       json
// @Produce      json
// @Param        item_id  path  int                       true  "交付文件ID"
// @Param        req      body  contract.ClientFeedbackReq true  "反馈内容"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /h5/delivery/feedback/{item_id} [post]
func (h *Controller) FeedbackSubmit(c *gin.Context) {
	cu := middleware.GetClientUser(c)
	itemID, err := bind.PathID(c, "item_id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req contract.ClientFeedbackReq
	if err := bind.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.ClientFeedbackSubmit(c.Request.Context(), cu, itemID, req); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}
