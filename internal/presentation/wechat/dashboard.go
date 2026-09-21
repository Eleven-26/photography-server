// 工作台（由 staff.go 按业务域拆分，2026-09-21 结构整改）。
package wechat

import (
	"photography-server/internal/middleware"
	"photography-server/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

// Overview 工作台待办统计
// @Summary      工作台待办
// @Description  员工端首页待办统计（今日拍摄 / 待处理事项等）。
// @Tags         员工端·工作台
// @Accept       json
// @Produce      json
// @Success      200  {object}  response.Body{data=contract.StaffOverview}
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /wechat/staff/overview [post]
func (h *Controller) Overview(c *gin.Context) {
	op := middleware.GetOperator(c)
	ov, err := h.Svc.StaffOverview(c.Request.Context(), op)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, ov)
}
