package controller

import (
	"github.com/gin-gonic/gin"

	"photography-server/internal/contract"
	"photography-server/internal/middleware"
	"photography-server/internal/pkg/params"
	"photography-server/internal/pkg/response"
	"photography-server/internal/presentation/bind"
)

// @Summary      档期列表
// @Description  按日期区间查询档期占用；photographer_id 可选，用于只看某位摄影师。
// @Tags         档期
// @Accept       json
// @Produce      json
// @Param        req  body  object{start_date=string,end_date=string,photographer_id=int}  true  "查询条件（日期格式 2006-01-02）"
// @Success      200  {object}  response.Body{data=[]model.CalendarBlock}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /calendar/list [post]
func (h *Controller) CalendarList(c *gin.Context) {
	op := middleware.GetOperator(c)
	photographerID := params.Int64(c, "photographer_id")
	list, err := h.Svc.ListCalendar(c.Request.Context(), op, bind.ParamStr(c, "start_date"), bind.ParamStr(c, "end_date"), photographerID)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, list)
}

// @Summary      锁定档期
// @Description  手动锁定某个时段（占用档期，防重复排期）。
// @Tags         档期
// @Accept       json
// @Produce      json
// @Param        req  body  contract.CalendarBlockReq  true  "档期信息（date / time_range 必填）"
// @Success      200  {object}  response.Body{data=model.CalendarBlock}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /calendar/lock [post]
func (h *Controller) CalendarLock(c *gin.Context) {
	op := middleware.GetOperator(c)
	var req contract.CalendarBlockReq
	if err := bind.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	block, err := h.Svc.BlockCalendar(c.Request.Context(), op, req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, block)
}

// @Summary      取消档期锁定
// @Description  释放已锁定的档期时段。
// @Tags         档期
// @Accept       json
// @Produce      json
// @Param        id  path  int  true  "档期记录ID"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /calendar/cancel/{id} [post]
func (h *Controller) CalendarCancel(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.CancelCalendarBlock(c.Request.Context(), op, id); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

// ---------------------------------------------------------------------
// 档期规则（排班时段模板）—— 与员工端共用同一 service，PC 端仅暴露入口
// ---------------------------------------------------------------------

// SlotTemplateList 档期时段模板列表
// @Summary      档期时段模板列表
// @Description  返回排班时段模板；photographer_id 可选，缺省取当前操作人。与员工端共用同一实现。
// @Tags         档期
// @Accept       json
// @Produce      json
// @Param        req  body  object{photographer_id=int}  false  "查询条件（可省略）"
// @Success      200  {object}  response.Body{data=[]model.SlotTemplate}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /calendar/slot-template/list [post]
func (h *Controller) SlotTemplateList(c *gin.Context) {
	op := middleware.GetOperator(c)
	list, err := h.Svc.SlotTemplates(c.Request.Context(), op, params.Int64(c, "photographer_id"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, list)
}

// SlotTemplateSave 新建（无 :id）/ 更新（有 :id）档期时段模板
// @Summary      保存档期时段模板
// @Description  新建（不传 :id）/ 更新（传 :id）档期时段模板；同一 handler 挂两个路径，与员工端共用实现。
// @Tags         档期
// @Accept       json
// @Produce      json
// @Param        id   path  int                            false  "模板ID（更新时传；新建不传）"
// @Param        req  body  contract.StaffSlotTemplateReq  true   "模板信息"
// @Success      200  {object}  response.Body{data=model.SlotTemplate}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /calendar/slot-template/save [post]
// @Router       /calendar/slot-template/save/{id} [post]
func (h *Controller) SlotTemplateSave(c *gin.Context) {
	op := middleware.GetOperator(c)
	var id int64
	if raw := c.Param("id"); raw != "" {
		parsed, err := bind.PathID(c, "id")
		if err != nil {
			response.Fail(c, err)
			return
		}
		id = parsed
	}
	var req contract.StaffSlotTemplateReq
	if err := bind.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	m, err := h.Svc.SaveSlotTemplate(c.Request.Context(), op, id, req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, m)
}

// SlotTemplateDelete 删除档期时段模板
// @Summary      删除档期时段模板
// @Description  删除排班时段模板。
// @Tags         档期
// @Accept       json
// @Produce      json
// @Param        id  path  int  true  "模板ID"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /calendar/slot-template/delete/{id} [post]
func (h *Controller) SlotTemplateDelete(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.DeleteSlotTemplate(c.Request.Context(), op, id); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}
