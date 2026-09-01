package enum

// UploadType 上传文件类型
type UploadType int

const (
	UploadTypeImage UploadType = 1 // 图片
	UploadTypeVideo UploadType = 2 // 视频
	UploadTypeFile  UploadType = 3 // 文件
)

var uploadTypeName = map[UploadType]string{
	UploadTypeImage: "图片",
	UploadTypeVideo: "视频",
	UploadTypeFile:  "文件",
}

func UploadTypeName(typ UploadType) string {
	if name, ok := uploadTypeName[typ]; ok {
		return name
	}
	return "未知"
}
