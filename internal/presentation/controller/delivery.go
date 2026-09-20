package controller

import (
	"github.com/gin-gonic/gin"

	"photography-server/internal/contract"
	"photography-server/internal/middleware"
	"photography-server/internal/pkg/response"
	"photography-server/internal/presentation/bind"
)

// DeliveryCreate 新建交付任务 body: DeliveryCreateReq（路径 :order_id 优先于 body.order_id）
// @Summary      创建交付任务
// @Description  为订单创建交付单；body.order_id 为订单主键（body 优先于路径参数）。
// @Tags         交付
// @Accept       json
// @Produce      json
// @Param        req  body  contract.DeliveryCreateReq  true  "交付信息（含 order_id）"
// @Success      200  {object}  response.Body{data=model.Delivery}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /delivery/create [post]
func (h *Controller) DeliveryCreate(c *gin.Context) {
	op := middleware.GetOperator(c)
	orderID, err := bind.BodyID(c, "order_id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req contract.DeliveryCreateReq
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
// @Summary      交付工作台列表
// @Description  分页查询交付单看板，按交付阶段筛选。
// @Tags         交付
// @Accept       json
// @Produce      json
// @Param        req  body  object{page=int,page_size=int,keyword=string,stage=int}  true  "查询条件（stage 见 enum.DeliveryStage，0/不传=全部）"
// @Success      200  {object}  response.Body{data=response.Page{list=[]repository.DeliveryListItem,total=int,page=int,page_size=int}}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /delivery/list [post]
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

// DeliveryRemind 提醒交付负责人（:id 为交付单 ID）
// @Summary      提醒交付负责人
// @Description  向交付单负责人推送提醒通知。⚠️ 本接口的 :id 是**交付单 ID**（非订单 ID）。
// @Tags         交付
// @Accept       json
// @Produce      json
// @Param        id  path  int  true  "交付单ID"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /delivery/remind/{id} [post]
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
// @Summary      交付单详情
// @Description  按订单反查交付单。⚠️ 本接口的 :id 是**订单 ID**（与 /delivery/remind 等推进类接口不同）。
// @Tags         交付
// @Accept       json
// @Produce      json
// @Param        id  path  int  true  "订单ID"
// @Success      200  {object}  response.Body{data=model.Delivery}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /delivery/detail/{id} [post]
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
// @Summary      交付文件明细
// @Description  按订单返回交付文件明细。⚠️ 本接口的 :id 是**订单 ID**，与 /delivery/detail/:id 语义一致。
// @Tags         交付
// @Accept       json
// @Produce      json
// @Param        id  path  int  true  "订单ID"
// @Success      200  {object}  response.Body{data=[]model.DeliveryItem}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /delivery/items/{id} [post]
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

// DeliveryUploadSamples 上传样片（:id 为交付单 ID）body: {items:[{url,...}]}
// @Summary      上传样片
// @Description  批量登记样片文件。⚠️ 本接口的 :id 是**交付单 ID**。
// @Tags         交付
// @Accept       json
// @Produce      json
// @Param        id   path  int     true  "交付单ID"
// @Param        req  body  object{items=[]contract.DeliveryItemReq}  true  "样片文件列表（至少 1 项）"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /delivery/upload-samples/{id} [post]
func (h *Controller) DeliveryUploadSamples(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req struct {
		Items []contract.DeliveryItemReq `json:"items" binding:"required"`
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

// DeliverySelect 客户选片（:id 为交付单 ID）body: {item_ids:[...]}
// @Summary      客户选片
// @Description  记录客户选片结果。⚠️ 本接口的 :id 是**交付单 ID**。
// @Tags         交付
// @Accept       json
// @Produce      json
// @Param        id   path  int                        true  "交付单ID"
// @Param        req  body  contract.DeliverySelectReq true  "选中的文件 ID 列表"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /delivery/select/{id} [post]
func (h *Controller) DeliverySelect(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req contract.DeliverySelectReq
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

// DeliveryUploadRetouched 上传精修成品（:id 为交付单 ID）body: {items:[...]}
// @Summary      上传精修成品
// @Description  批量登记精修成品文件（进入待客户确认）。⚠️ 本接口的 :id 是**交付单 ID**。
// @Tags         交付
// @Accept       json
// @Produce      json
// @Param        id   path  int     true  "交付单ID"
// @Param        req  body  object{items=[]contract.DeliveryItemReq}  true  "精修文件列表（至少 1 项）"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /delivery/upload-retouched/{id} [post]
func (h *Controller) DeliveryUploadRetouched(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req struct {
		Items []contract.DeliveryItemReq `json:"items" binding:"required"`
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

// DeliveryConfirm 标记交付完成（:id 为交付单 ID）
// @Summary      标记交付完成
// @Description  把交付单置为已交付。⚠️ 本接口的 :id 是**交付单 ID**。
// @Tags         交付
// @Accept       json
// @Produce      json
// @Param        id  path  int  true  "交付单ID"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /delivery/confirm/{id} [post]
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
