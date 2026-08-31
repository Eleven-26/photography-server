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

// 退款比例（按拍摄前小时数）
const (
	RefundRatioFull = 1.0 // >=72h 全额
	RefundRatio80   = 0.8 // >=48h 退80%
	RefundRatio50   = 0.5 // >=24h 退50%
	RefundRatioNone = 0   // <24h 不退
)
