// 登录设备（由 staff.go 按业务域拆分，2026-09-21 结构整改）。
package wechat

import (
	"photography-server/internal/middleware"
	"photography-server/internal/pkg/response"
	"photography-server/internal/presentation/bind"

	"github.com/gin-gonic/gin"
)

// DeviceList 登录设备列表
// @Summary      登录设备列表
// @Description  当前账号的登录设备（个人中心 → 设备管理；操作对象是登录者本人，免权限点）。
// @Tags         员工端·个人中心
// @Accept       json
// @Produce      json
// @Success      200  {object}  response.Body{data=[]model.UserDevice}
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /wechat/staff/device/list [post]
func (h *Controller) DeviceList(c *gin.Context) {
	op := middleware.GetOperator(c)
	list, err := h.Svc.ListDevices(c.Request.Context(), op)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, list)
}

// DeviceRemove 踢出登录设备
// @Summary      踢出登录设备
// @Description  强制下线某台登录设备（免权限点：操作对象是登录者本人设备）。
// @Tags         员工端·个人中心
// @Accept       json
// @Produce      json
// @Param        id  path  int  true  "设备ID"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /wechat/staff/device/remove/{id} [post]
func (h *Controller) DeviceRemove(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.RemoveDevice(c.Request.Context(), op, id); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}
