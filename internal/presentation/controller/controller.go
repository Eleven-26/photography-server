package controller

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"photography-server/internal/common"
	"photography-server/internal/config"
	"photography-server/internal/pkg/errs"
	"photography-server/internal/pkg/params"
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
		return errs.BadRequest(errs.ErrBadRequest + "：" + err.Error())
	}
	return nil
}

// bearerToken 从 Authorization: Bearer 头取令牌（登出/吊销等场景用）
func bearerToken(c *gin.Context) string {
	auth := c.GetHeader("Authorization")
	if len(auth) > 7 && auth[:7] == "Bearer " {
		return auth[7:]
	}
	return ""
}

func pager(c *gin.Context) (int, int) {
	page := params.Int(c, "page")
	pageSize := params.Int(c, "page_size")
	if page <= 0 {
		page = common.DefaultPage
	}
	if pageSize <= 0 {
		pageSize = common.DefaultPageSize
	}
	if pageSize > common.MaxPageSize {
		pageSize = common.MaxPageSize
	}
	return page, pageSize
}

// queryStr 取字符串参数：统一从 POST body 取（见 pkg/params），不再读 query
func queryStr(c *gin.Context, key string) string {
	return params.Str(c, key)
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
