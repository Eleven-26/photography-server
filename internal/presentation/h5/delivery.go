// 选片与交付（由 h5.go 按业务域拆分，2026-09-21 结构整改）。
package h5

import (
	"photography-server/internal/contract"
	"photography-server/internal/middleware"
	"photography-server/internal/pkg/response"
	"photography-server/internal/presentation/bind"

	"github.com/gin-gonic/gin"
)

// DeliveryDetail 交付单与明细（选片页/成片页）。:id 为 **order_id**（与 PC 端同语义）。
// @Summary      交付单与明细
// @Description  选片页 / 成片页数据源。⚠️ 本接口的 :id 是 **order_id**（与 PC 端同语义）。
// @Tags         客户区·交付
// @Accept       json
// @Produce      json
// @Param        id  path  int  true  "订单ID"
// @Success      200  {object}  response.Body{data=object{delivery=model.Delivery,items=[]model.DeliveryItem}}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /h5/delivery/detail/{id} [post]
func (h *Controller) DeliveryDetail(c *gin.Context) {
	cu := middleware.GetClientUser(c)
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	d, items, err := h.Svc.ClientDeliveryDetail(c.Request.Context(), cu, id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{"delivery": d, "items": items})
}

// DeliveryItems 交付文件明细（按订单反查）。:id 为 **order_id**。
// 与 /delivery/detail/:id 数据同源，供「文件管理」tab 直接取列表；未建交付单返回空列表。
// @Summary      交付文件明细
// @Description  按订单反查交付文件明细，供「文件管理」tab 直接取列表；未建交付单返回空列表。⚠️ :id 是 **order_id**。
// @Tags         客户区·交付
// @Accept       json
// @Produce      json
// @Param        id  path  int  true  "订单ID"
// @Success      200  {object}  response.Body{data=[]model.DeliveryItem}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /h5/delivery/items/{id} [post]
func (h *Controller) DeliveryItems(c *gin.Context) {
	cu := middleware.GetClientUser(c)
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	_, items, err := h.Svc.ClientDeliveryDetail(c.Request.Context(), cu, id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, items)
}

// SelectPhotos 提交选片
// @Summary      提交选片
// @Description  ⚠️ 本接口的 :id 是**交付单 ID**。
// @Tags         客户区·交付
// @Accept       json
// @Produce      json
// @Param        id   path  int                      true  "交付单ID"
// @Param        req  body  object{item_ids=[]int64} true  "选中的文件 ID 列表"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /h5/delivery/select/{id} [post]
func (h *Controller) SelectPhotos(c *gin.Context) {
	cu := middleware.GetClientUser(c)
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req struct {
		ItemIDs []int64 `json:"item_ids" binding:"required"`
	}
	if err := bind.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.ClientSelectPhotos(c.Request.Context(), cu, id, req.ItemIDs); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

// ConfirmExtra 确认加片费用
// @Summary      确认加片费用
// @Description  ⚠️ 本接口的 :id 是**交付单 ID**。
// @Tags         客户区·交付
// @Accept       json
// @Produce      json
// @Param        id  path  int  true  "交付单ID"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /h5/delivery/confirm-extra/{id} [post]
func (h *Controller) ConfirmExtra(c *gin.Context) {
	cu := middleware.GetClientUser(c)
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.ClientConfirmExtra(c.Request.Context(), cu, id); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

// ConfirmDelivery 确认成片
// @Summary      确认成片
// @Description  ⚠️ 本接口的 :id 是**交付单 ID**。
// @Tags         客户区·交付
// @Accept       json
// @Produce      json
// @Param        id  path  int  true  "交付单ID"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /h5/delivery/confirm/{id} [post]
func (h *Controller) ConfirmDelivery(c *gin.Context) {
	cu := middleware.GetClientUser(c)
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.ClientConfirmDelivery(c.Request.Context(), cu, id); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

// ExtraQuote 加片费试算（body 可为空：按当前已选张数试算）
// @Summary      加片费试算
// @Description  按已选张数试算加片费；**body 可为空**（缺省按当前已选张数试算）。⚠️ :id 是**交付单 ID**。
// @Tags         客户区·交付
// @Accept       json
// @Produce      json
// @Param        id   path  int                            true  "交付单ID"
// @Param        req  body  contract.ClientExtraQuoteReq    false  "试算参数（可省略）"
// @Success      200  {object}  response.Body{data=contract.ClientExtraQuoteResp}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /h5/delivery/extra-quote/{id} [post]
func (h *Controller) ExtraQuote(c *gin.Context) {
	cu := middleware.GetClientUser(c)
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req contract.ClientExtraQuoteReq
	_ = c.ShouldBindJSON(&req)
	q, err := h.Svc.ClientExtraQuote(c.Request.Context(), cu, id, req.SelectCount)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, q)
}
