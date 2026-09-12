package controller

import (
	"github.com/gin-gonic/gin"

	"photography-server/internal/middleware"
	"photography-server/internal/presentation/bind"
	"photography-server/internal/presentation/dto"
	"photography-server/internal/presentation/response"
)

// DeliveryCreate 新建交付任务 body: DeliveryCreateReq（路径 :order_id 优先于 body.order_id）
func (h *Controller) DeliveryCreate(c *gin.Context) {
	op := middleware.GetOperator(c)
	orderID, err := bind.PathID(c, "order_id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req dto.DeliveryCreateReq
	if err := bind.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	req.OrderID = orderID
	d, err := h.Svc.CreateDeliveryTask(c.Request.Context(), op, req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, d)
}

// DeliveryList 交付工作台看板列表（按阶段筛选）
func (h *Controller) DeliveryList(c *gin.Context) {
	op := middleware.GetOperator(c)
	page, pageSize := bind.Pager(c)
	list, total, err := h.Svc.ListDeliveries(c.Request.Context(), op, bind.ParamInt(c, "stage"), page, pageSize, bind.ParamStr(c, "keyword"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.PageOK(c, list, total, page, pageSize)
}

// DeliveryRemind 提醒交付负责人
func (h *Controller) DeliveryRemind(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.RemindDeliveryOperator(c.Request.Context(), op, id); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

// DeliveryDetail 交付单详情（:id 为 order_id）
func (h *Controller) DeliveryDetail(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	d, err := h.Svc.GetDeliveryByOrder(c.Request.Context(), op, id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, d)
}

// DeliveryItems 交付文件明细（:id 为 order_id，与 /delivery/detail/:id 语义一致）
func (h *Controller) DeliveryItems(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	list, err := h.Svc.ListDeliveryItemsByOrder(c.Request.Context(), op, id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, list)
}

// DeliveryUploadSamples 上传样片 body: {items:[{url,...}]}
func (h *Controller) DeliveryUploadSamples(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req struct {
		Items []dto.DeliveryItemReq `json:"items" binding:"required"`
	}
	if err := bind.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.UploadSamples(c.Request.Context(), op, id, req.Items); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

// DeliverySelect 客户选片 body: {item_ids:[...]}
func (h *Controller) DeliverySelect(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req dto.DeliverySelectReq
	if err := bind.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.SelectPhotos(c.Request.Context(), op, id, req); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

// DeliveryUploadRetouched 上传精修成品 body: {items:[...]}
func (h *Controller) DeliveryUploadRetouched(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req struct {
		Items []dto.DeliveryItemReq `json:"items" binding:"required"`
	}
	if err := bind.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.UploadRetouched(c.Request.Context(), op, id, req.Items); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

func (h *Controller) DeliveryConfirm(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.ConfirmDelivered(c.Request.Context(), op, id); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}
