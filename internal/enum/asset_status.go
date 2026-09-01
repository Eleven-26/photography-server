package enum

// AssetStatus 作品状态
type AssetStatus int

const (
	AssetStatusDraft     AssetStatus = 1 // 草稿
	AssetStatusPublished AssetStatus = 2 // 已发布
)

var assetStatusName = map[AssetStatus]string{
	AssetStatusDraft:     "草稿",
	AssetStatusPublished: "已发布",
}

func AssetStatusName(status AssetStatus) string {
	if name, ok := assetStatusName[status]; ok {
		return name
	}
	return "未知"
}
