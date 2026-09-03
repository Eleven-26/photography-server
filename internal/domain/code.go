package domain

import (
	"fmt"
	"math/rand"
	"time"
)

// GenCode 生成业务编号，如 SL-260817-1234（前缀-年月日-4位随机数）
func GenCode(prefix string) string {
	return fmt.Sprintf("%s-%s-%04d", prefix, time.Now().Format("060102"), rand.Intn(10000))
}
