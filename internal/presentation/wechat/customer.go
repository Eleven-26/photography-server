// 客户档案（由 staff.go 按业务域拆分，2026-09-21 结构整改）。
package wechat

import (
	"photography-server/internal/contract"
	"photography-server/internal/middleware"
	"photography-server/internal/pkg/params"
	"photography-server/internal/pkg/response"
	"photography-server/internal/presentation/bind"

	"github.com/gin-gonic/gin"
)

// TodayFollow 今日待跟进（到期/逾期且未成交未流失的线索）
// @Summary      今日待跟进
// @Description  返回到期/逾期且未成交未流失的线索（客户档案页今日待跟进）。跟进对象是线索。
// @Tags         员工端·客户档案
// @Accept       json
// @Produce      json
// @Param        req  body  object{limit=int}  true  "返回条数上限（0=默认）"
// @Success      200  {object}  response.Body{data=[]model.Lead}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /wechat/staff/customer/today-follow [post]
func (h *Controller) TodayFollow(c *gin.Context) {
	op := middleware.GetOperator(c)
	list, err := h.Svc.StaffTodayFollowUp(c.Request.Context(), op, params.Int(c, "limit"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, list)
}

// CustomerMobileUpdate 修改客户手机号（换绑，含格式与占用校验）
// @Summary      修改客户手机号
// @Description  为客户换绑手机号（含格式校验与租户内占用校验）。
// @Tags         员工端·客户档案
// @Accept       json
// @Produce      json
// @Param        req  body  contract.StaffCustomerMobileReq  true  "客户ID与新手机号"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /wechat/staff/customer/mobile [post]
func (h *Controller) CustomerMobileUpdate(c *gin.Context) {
	op := middleware.GetOperator(c)
	var req contract.StaffCustomerMobileReq
	if err := bind.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.UpdateCustomerMobile(c.Request.Context(), op, req.CustomerID, req.Mobile); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}
