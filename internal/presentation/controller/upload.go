package controller

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"photography-server/internal/middleware"
	"photography-server/internal/pkg/errs"
	"photography-server/internal/response"
)

// UploadFile 图片/视频上传（multipart/form-data: file 字段）
func (h *Controller) UploadFile(c *gin.Context) {
	op := middleware.GetOperator(c)
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		response.Fail(c, errs.BadRequest("请上传文件"))
		return
	}
	defer file.Close()

	if h.Cfg.Upload.MaxSizeMB > 0 && header.Size > int64(h.Cfg.Upload.MaxSizeMB)*1024*1024 {
		response.Fail(c, errs.BadRequest("文件大小超过限制"))
		return
	}

	buf := make([]byte, header.Size)
	if _, err := file.Read(buf); err != nil {
		response.Fail(c, errs.BadRequest("文件读取失败"))
		return
	}

	storeID, _ := strconv.ParseInt(c.PostForm("store_id"), 10, 64)
	bizID, _ := strconv.ParseInt(c.PostForm("biz_id"), 10, 64)
	result, err := h.Svc.UploadFile(op, storeID, c.PostForm("biz_type"), bizID, header.Filename, buf)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, result)
}
