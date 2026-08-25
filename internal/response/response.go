package response

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"photography-server/internal/pkg/errs"
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

func Fail(c *gin.Context, err error) {
	be, ok := err.(*errs.BizError)
	if !ok {
		be = errs.Internal(err.Error())
	}
	c.JSON(errs.HTTPStatus(err), Body{Code: be.Code, Msg: be.Msg, Data: nil})
}
