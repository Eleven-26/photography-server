package enum

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
