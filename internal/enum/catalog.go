package enum

// 本文件聚合「套餐 / 档期 / 作品集」内容与商品枚举。
// 原拆分为 package_status.go / block_status.go / asset_status.go（#43 合并）。

// PackageStatus 套餐状态
type PackageStatus int

const (
	PackageStatusDraft   PackageStatus = 1 // 草稿
	PackageStatusActive  PackageStatus = 2 // 已上架
	PackageStatusOffline PackageStatus = 3 // 已下线
)

var packageStatusName = map[PackageStatus]string{
	PackageStatusDraft:   "草稿",
	PackageStatusActive:  "已上架",
	PackageStatusOffline: "已下线",
}

func PackageStatusName(status PackageStatus) string {
	if name, ok := packageStatusName[status]; ok {
		return name
	}
	return "未知"
}

// BlockStatus 档期锁定状态
type BlockStatus int

const (
	BlockStatusLocked    BlockStatus = 1 // 已锁定
	BlockStatusCancelled BlockStatus = 2 // 已取消
)

var blockStatusName = map[BlockStatus]string{
	BlockStatusLocked:    "已锁定",
	BlockStatusCancelled: "已取消",
}

func BlockStatusName(status BlockStatus) string {
	if name, ok := blockStatusName[status]; ok {
		return name
	}
	return "未知"
}

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
