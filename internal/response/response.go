package response

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"photography-server/internal/pkg/errs"
	"photography-server/internal/pkg/logger"
)

type Body struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

type Page struct {
	List     interface{} `json:"list"`
	Total    int64       `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"page_size"`
}

func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Body{Code: 0, Msg: "ok", Data: data})
}

func OKNil(c *gin.Context) {
	OK(c, nil)
}

func PageOK(c *gin.Context, list interface{}, total int64, page, pageSize int) {
	OK(c, Page{List: list, Total: total, Page: page, PageSize: pageSize})
}

// Fail 统一错误响应。业务错误（*errs.BizError）按定义透出；
// 非业务错误（内部异常）不回显内部细节，仅返回通用文案，完整错误记入服务端日志。
func Fail(c *gin.Context, err error) {
	be, ok := err.(*errs.BizError)
	if !ok {
		logger.Errorf("[response] internal error: path=%s err=%v", c.Request.URL.Path, err)
		be = errs.Internal("") // 默认文案：系统繁忙，请稍后再试
	}
	c.JSON(errs.HTTPStatus(err), Body{Code: be.Code, Msg: be.Msg, Data: nil})
}
