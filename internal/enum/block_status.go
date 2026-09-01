package enum

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
