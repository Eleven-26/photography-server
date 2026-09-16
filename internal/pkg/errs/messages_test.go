package errs

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

// reInlineMessage 匹配 errs.X("非空文案") —— 也就是「把错误提示写死在调用点」的写法。
// 捕获组 1 是文案本身；errs.X("") 是有意传空以使用默认文案，不算违规。
var reInlineMessage = regexp.MustCompile(`errs\.[A-Za-z]+\(\s*"([^"]*)"\s*\)`)

// TestNoInlineErrorMessages 全仓护栏：错误文案只能来自本包的常量。
//
// 背景：项目约定所有错误提示集中在 errs/messages.go，调用点写 errs.BadRequest(errs.ErrXxx)。
// 但"顺手写个字面量"太容易，散落开以后改文案要全仓翻，前端也没法按文案做映射。
// 这个测试把约定钉死：任何 errs.X("非空字符串") 一律失败。
//
// 有意排除的写法：
//   - 传空串（使用该错误类型的默认文案）；
//   - 已引用常量的写法；
//   - 固定前缀与运行时详情拼接（参数不是纯字面量，不匹配上面的正则）。
func TestNoInlineErrorMessages(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("无法定位测试文件路径")
	}
	root := filepath.Join(filepath.Dir(thisFile), "..", "..") // internal/pkg/errs -> internal
	self := filepath.Clean(thisFile)

	var offences []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		// 跳过本文件（注释里含着反例写法）与常量定义文件
		if strings.EqualFold(filepath.Clean(path), self) || filepath.Base(path) == "messages.go" {
			return nil
		}
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		for i, line := range strings.Split(string(raw), "\n") {
			for _, m := range reInlineMessage.FindAllStringSubmatch(line, -1) {
				if strings.TrimSpace(m[1]) == "" {
					continue // 有意传空
				}
				rel, _ := filepath.Rel(root, path)
				offences = append(offences, filepath.ToSlash(rel)+":"+strconv.Itoa(i+1)+"  "+strings.TrimSpace(line))
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("遍历 internal 失败：%v", err)
	}

	if len(offences) > 0 {
		t.Errorf("发现 %d 处硬编码错误文案，请改用 errs.ErrXxx 常量：\n%s",
			len(offences), strings.Join(offences, "\n"))
	}
}
