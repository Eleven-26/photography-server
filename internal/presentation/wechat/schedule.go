// 日程（由 staff.go 按业务域拆分，2026-09-21 结构整改）。
package wechat

import (
	"photography-server/internal/middleware"
	"photography-server/internal/pkg/params"
	"photography-server/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

// ScheduleList 日程列表（body: start_date/end_date/photographer_id）
// @Summary      日程列表
// @Description  员工端日程（按日期区间查询档期占用）；复用管理端 ListCalendar。
// @Tags         员工端·日程
// @Accept       json
// @Produce      json
// @Param        req  body  object{start_date=string,end_date=string,photographer_id=int}  true  "查询条件（日期格式 2006-01-02）"
// @Success      200  {object}  response.Body{data=[]model.CalendarBlock}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /wechat/staff/schedule/list [post]
func (h *Controller) ScheduleList(c *gin.Context) {
	op := middleware.GetOperator(c)
	photographerID := params.Int64(c, "photographer_id")
	list, err := h.Svc.ListCalendar(c.Request.Context(), op, params.Str(c, "start_date"), params.Str(c, "end_date"), photographerID)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, list)
}
