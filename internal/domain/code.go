package domain

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// GenCode 生成业务编号，如 SL-260817-1a2b3c4d（前缀-年月日-8位十六进制随机串）。
// 随机位使用 crypto/rand（4 字节），同日同前缀碰撞概率 ~1/2^32，相比原 4 位随机数（1/10^4）
// 提升约 42 万倍，基本消除唯一索引冲突；仍以数据库唯一索引为最终兜底。
func GenCode(prefix string) string {
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		// crypto/rand 不可用时退化为时间纳秒低位，保证函数不失败
		return fmt.Sprintf("%s-%s-%08x", prefix, time.Now().Format("060102"), time.Now().UnixNano()&0xffffffff)
	}
	return fmt.Sprintf("%s-%s-%s", prefix, time.Now().Format("060102"), hex.EncodeToString(b[:]))
}
