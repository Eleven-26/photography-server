package controller

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"photography-server/internal/config"
	"photography-server/internal/pkg/errs"
	"photography-server/internal/service"
)

// Controller 所有接口处理器统一挂在 Controller 上
type Controller struct {
	Svc *service.Service
	Cfg *config.Config
}

func New(svc *service.Service, cfg *config.Config) *Controller {
	return &Controller{Svc: svc, Cfg: cfg}
}

// bindJSON 绑定 JSON 请求体
func (h *Controller) bindJSON(c *gin.Context, obj interface{}) error {
	if err := c.ShouldBindJSON(obj); err != nil {
		return errs.BadRequest("请求参数错误：" + err.Error())
	}
	return nil
}

// bindQuery 绑定查询参数
func (h *Controller) bindQuery(c *gin.Context, obj interface{}) error {
	if err := c.ShouldBindQuery(obj); err != nil {
		return errs.BadRequest("查询参数错误：" + err.Error())
	}
	return nil
}

func pager(c *gin.Context) (int, int) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 200 {
		pageSize = 20
	}
	return page, pageSize
}

func queryStr(c *gin.Context, key string) string {
	return c.Query(key)
}

func pathID(c *gin.Context) (int64, error) {
	return pathParam(c, "id")
}

func pathParam(c *gin.Context, name string) (int64, error) {
	id, err := strconv.ParseInt(c.Param(name), 10, 64)
	if err != nil || id <= 0 {
		return 0, errs.BadRequest("参数错误")
	}
	return id, nil
}
