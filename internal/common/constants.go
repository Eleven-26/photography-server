package common

// 响应码
const (
	CodeOK           = 0
	CodeBadRequest   = 400
	CodeUnauthorized = 401
	CodeForbidden    = 403
	CodeNotFound     = 404
	CodeInternal     = 500
)

// 分页默认值
const (
	DefaultPage     = 1
	DefaultPageSize = 20
	MaxPageSize     = 200
)

// 文件上传
const (
	MaxUploadSizeMB = 100
)
