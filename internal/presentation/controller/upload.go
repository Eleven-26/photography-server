package controller

import (
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"photography-server/internal/middleware"
	"photography-server/internal/pkg/errs"
	"photography-server/internal/presentation/response"
)

// UploadFile 图片/视频上传（multipart/form-data: file 字段）
// 安全要点：
//  1. 用 http.MaxBytesReader 在 Gin 解析 multipart 之前限制请求体大小，
//     避免超大文件先进内存/临时文件后才发现超限（旧实现限制形同虚设）；
//  2. 用 io.ReadAll 完整读取，避免单次 Read 只读一个 chunk 导致大文件被截断。
func (h *Controller) UploadFile(c *gin.Context) {
	op := middleware.GetOperator(c)

	if h.Cfg.Upload.MaxSizeMB > 0 {
		// 请求体 = multipart 信封 + 文件内容，留出 1MB 信封余量
		limit := int64(h.Cfg.Upload.MaxSizeMB)*1024*1024 + 1<<20
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limit)
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		// MaxBytesError 说明请求体超限（可能被 io.ErrUnexpectedEOF 包装，用 errors.As 兜底）
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			response.Fail(c, errs.BadRequest("文件大小超过限制"))
			return
		}
		response.Fail(c, errs.BadRequest("请上传文件"))
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			response.Fail(c, errs.BadRequest("文件大小超过限制"))
			return
		}
		response.Fail(c, errs.BadRequest("文件读取失败"))
		return
	}
	if h.Cfg.Upload.MaxSizeMB > 0 && int64(len(data)) > int64(h.Cfg.Upload.MaxSizeMB)*1024*1024 {
		response.Fail(c, errs.BadRequest("文件大小超过限制"))
		return
	}

	storeID, _ := strconv.ParseInt(c.PostForm("store_id"), 10, 64)
	bizID, _ := strconv.ParseInt(c.PostForm("biz_id"), 10, 64)
	result, err := h.Svc.UploadFile(c.Request.Context(), op, storeID, c.PostForm("biz_type"), bizID, header.Filename, data)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, result)
}
