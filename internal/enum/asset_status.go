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

// AssetVisibility 作品可见性
const (
	AssetVisibilityPublic  = 1 // 公开（客户 H5 可见）
	AssetVisibilityPrivate = 2 // 未公开（仅工作室内部可见）
)

// AssetAuthorization 客户授权状态
const (
	AssetAuthPending = 1 // 待授权
	AssetAuthGranted = 2 // 已授权
)
