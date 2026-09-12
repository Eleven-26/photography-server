// Package bind 提供各端共用的请求参数绑定与分页辅助。
//
// 背景（2026-09-12 路由整理）：bindJSON / pager / pathID 曾在 controller、h5、wechat
// 三处各复制一份，且签名不一致——admin 版 pathID 把 "id" 硬编码在函数体内
// （路由参数一旦改名会静默失效：c.Param 返回空串 → ParseInt 失败 → 400，看起来像业务错误）。
// 现统一为显式参数的单一实现，各端只保留调用、不留副本。
package bind

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"photography-server/internal/common"
	"photography-server/internal/pkg/errs"
	"photography-server/internal/pkg/params"
)

// BindJSON 绑定 JSON 请求体，失败时返回统一的参数错误。
func BindJSON(c *gin.Context, obj interface{}) error {
	if err := c.ShouldBindJSON(obj); err != nil {
		return errs.BadRequest(errs.ErrBadRequest + "：" + err.Error())
	}
	return nil
}

// Pager 取分页参数，统一走项目约定（common.DefaultPage / DefaultPageSize / MaxPageSize）。
//
// 归一化规则：page<=0 → 1；pageSize<=0 → 默认值；pageSize> 上限 → **截断到上限**。
//
// ⚠️ 行为变更（2026-09-12）：h5 / wechat 端历史上各自硬编码 10 / 100，
// 且超限时是"重置为 10"而非"截断到上限"。统一后两端默认页大小 10 → 20、上限 100 → 200。
// 前端显式传 page_size 时无差异；不传时返回条数变多（不会变少）。
func Pager(c *gin.Context) (int, int) {
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

// PathID 取路径参数并解析为正整数 ID。name 必须与路由中的 :name 一致（显式传入，不硬编码）。
func PathID(c *gin.Context, name string) (int64, error) {
	id, err := strconv.ParseInt(c.Param(name), 10, 64)
	if err != nil || id <= 0 {
		return 0, errs.BadRequest("参数错误")
	}
	return id, nil
}

// ParamStr 取字符串参数：统一从 POST body 取（预解析见 pkg/params），不读 query。
func ParamStr(c *gin.Context, key string) string { return params.Str(c, key) }

// ParamInt 取整型参数：统一从 POST body 取。
func ParamInt(c *gin.Context, key string) int { return params.Int(c, key) }

// ParamInt64 取 64 位整型参数：统一从 POST body 取。
func ParamInt64(c *gin.Context, key string) int64 { return params.Int64(c, key) }
