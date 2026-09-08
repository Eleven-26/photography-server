// Command configctl 配置敏感值加解密工具（方案 B：字段级 ENC 密文）
//
// 主密钥（KEK）只从 APP_CONFIG_SECRET_FILE / APP_CONFIG_SECRET 读取，不进 Nacos、不进 git。
//
// 用法:
//
//	configctl keygen                       生成主密钥（base64，32 字节）
//	configctl encrypt -v '明文'             输出 ENCv1: 密文（也可 echo '明文' | configctl encrypt）
//	configctl decrypt -v 'ENCv1:...'        输出明文（排障用）
//	configctl check -f config/nacos/x.yaml  检查模板里是否残留明文敏感值（CI 可用）
package main

import (
	"bufio"
	"crypto/rand"
	"encoding/base64"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"

	"photography-server/internal/config"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "keygen":
		err = keygen()
	case "encrypt":
		err = transform(os.Args[2:], true)
	case "decrypt":
		err = transform(os.Args[2:], false)
	case "check":
		err = check(os.Args[2:])
	case "encrypt-file":
		err = encryptFile(os.Args[2:])
	default:
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "❌ "+err.Error())
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `用法:
  configctl keygen                        生成主密钥（base64，32 字节）
  configctl encrypt -v '明文'              输出 ENCv1: 密文（或 echo '明文' | configctl encrypt）
  configctl decrypt -v 'ENCv1:...'         输出明文（排障用）
  configctl check -f config/nacos/x.yaml   检查模板是否残留明文敏感值
  configctl encrypt-file -f config/nacos/x.yaml [-o out.yaml] [--dry-run]
                                           按敏感字段清单批量加密模板（保留注释与格式）

主密钥来源（优先级从高到低）：
  APP_CONFIG_SECRET_FILE   密钥文件路径（Docker/K8s secret 挂载，推荐）
  APP_CONFIG_SECRET        环境变量（base64 44 字符 / hex 64 字符 / 原文 32 字符）
  APP_CONFIG_SECRET_PREV   轮换期旧密钥（只用于解密）
`)
}

func keygen() error {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return err
	}
	fmt.Println(base64.StdEncoding.EncodeToString(b))
	fmt.Fprintln(os.Stderr, "把上一行写入 .env 的 APP_CONFIG_SECRET（本地），或写入 secret 文件并设 APP_CONFIG_SECRET_FILE 指向它；切勿提交 git。")
	return nil
}

func transform(args []string, encrypt bool) error {
	fs := flag.NewFlagSet("configctl", flag.ExitOnError)
	val := fs.String("v", "", "待处理的字符串；为空则从 stdin 读取")
	_ = fs.Parse(args)

	in := *val
	if in == "" {
		sc := bufio.NewScanner(os.Stdin)
		if sc.Scan() {
			in = sc.Text()
		}
	}
	in = strings.TrimRight(in, "\r\n")
	if in == "" {
		return fmt.Errorf("缺少输入：用 -v '值' 或管道传入")
	}

	c, err := config.LoadCipher()
	if err != nil {
		return err
	}
	if !c.Enabled() {
		return fmt.Errorf("未配置主密钥：先执行 configctl keygen，再设置 APP_CONFIG_SECRET（或 APP_CONFIG_SECRET_FILE）")
	}
	if encrypt {
		out, err := c.Encrypt(in)
		if err != nil {
			return err
		}
		fmt.Println(out)
		return nil
	}
	out, err := c.Decrypt(in)
	if err != nil {
		return err
	}
	fmt.Println(out)
	return nil
}

// 敏感键判定：高危（阻断）/ 低危（提示）
var (
	highRiskKeys = []string{"password", "passwd", "pwd", "secret", "token", "api_key", "apikey", "private_key"}
	lowRiskKeys  = []string{"username", "user"}
	kvRe         = regexp.MustCompile(`^(\s*)([A-Za-z0-9_.\-]+)\s*:\s*(.*)$`)
	sectionRe    = regexp.MustCompile(`^([A-Za-z0-9_\-]+):\s*(#.*)?$`) // 顶层配置段
)

// isSensitive 判断某段某键是否属于需要加密的敏感项
// （含 mongodb.uri 内嵌凭据场景：uri 里常直接写 mongodb://user:pass@host）
func isSensitive(section, key string) bool {
	k := strings.ToLower(key)
	for _, s := range highRiskKeys {
		if strings.Contains(k, s) {
			return true
		}
	}
	switch strings.ToLower(section) {
	case "db":
		return k == "user"
	case "mongodb":
		return k == "uri" || k == "username"
	case "elasticsearch":
		return k == "username"
	}
	return false
}

// encryptFile 批量加密模板中的敏感值：只替换值，保留缩进、行内注释与整体格式
func encryptFile(args []string) error {
	fs := flag.NewFlagSet("encrypt-file", flag.ExitOnError)
	file := fs.String("f", "", "待加密的 yaml 模板")
	out := fs.String("o", "", "输出文件（默认原地覆盖）")
	dry := fs.Bool("dry-run", false, "只列出将被加密的字段，不写文件")
	_ = fs.Parse(args)
	if *file == "" {
		return fmt.Errorf("未指定文件：-f <yaml>")
	}

	c, err := config.LoadCipher()
	if err != nil {
		return err
	}
	if !c.Enabled() {
		return fmt.Errorf("未配置主密钥：先执行 configctl keygen，再设置 APP_CONFIG_SECRET（或 APP_CONFIG_SECRET_FILE）")
	}

	raw, err := os.ReadFile(*file)
	if err != nil {
		return err
	}
	crlf := strings.Contains(string(raw), "\r\n")
	lines := strings.Split(strings.ReplaceAll(string(raw), "\r\n", "\n"), "\n")

	section := ""
	changed := 0
	for i, line := range lines {
		if m := sectionRe.FindStringSubmatch(line); m != nil {
			section = m[1]
			continue
		}
		idx := kvRe.FindStringSubmatchIndex(line)
		if idx == nil {
			continue
		}
		key := line[idx[4]:idx[5]]
		valStart, valEnd := idx[6], idx[7]
		rawVal := line[valStart:valEnd]
		val := strings.Trim(strings.TrimSpace(stripComment(rawVal)), `"'`)
		if val == "" || strings.HasPrefix(val, config.EncPrefix) {
			continue // 空值（走 env 注入）或已是密文
		}
		if !isSensitive(section, key) {
			continue
		}
		cipherText, err := c.Encrypt(val)
		if err != nil {
			return fmt.Errorf("加密 %s.%s 失败: %w", section, key, err)
		}
		comment := ""
		if ci := commentIndex(rawVal); ci >= 0 {
			comment = " " + strings.TrimSpace(rawVal[ci:])
		}
		lines[i] = line[:valStart] + `"` + cipherText + `"` + comment
		changed++
		fmt.Fprintf(os.Stderr, "  %s.%s → 已加密\n", section, key)
	}

	joined := strings.Join(lines, "\n")
	if crlf {
		joined = strings.ReplaceAll(joined, "\n", "\r\n")
	}
	if *dry {
		fmt.Printf("（--dry-run）%s：将加密 %d 处，未写入\n", *file, changed)
		return nil
	}
	target := *file
	if *out != "" {
		target = *out
	}
	if err := os.WriteFile(target, []byte(joined), 0o644); err != nil {
		return err
	}
	fmt.Printf("已加密 %s：%d 处敏感值 → %s\n", *file, changed, target)
	return nil
}

// stripComment 去掉 yaml 行内注释（引号内的 # 不处理）
func stripComment(s string) string {
	if i := commentIndex(s); i >= 0 {
		return strings.TrimSpace(s[:i])
	}
	return strings.TrimSpace(s)
}

// commentIndex 返回行内注释起始下标，无注释返回 -1
func commentIndex(s string) int {
	inQuote := false
	var quote byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		if inQuote {
			if c == quote {
				inQuote = false
			}
			continue
		}
		switch c {
		case '"', '\'':
			inQuote, quote = true, c
		case '#':
			if i == 0 || s[i-1] == ' ' || s[i-1] == '\t' {
				return i
			}
		}
	}
	return -1
}

func check(args []string) error {
	fs := flag.NewFlagSet("check", flag.ExitOnError)
	file := fs.String("f", "", "待检查的 yaml 文件")
	warnOnly := fs.Bool("warn-only", false, "明文敏感值只提示不阻断（dev 模板刻意保留明文时使用）")
	_ = fs.Parse(args)
	if *file == "" {
		return fmt.Errorf("未指定文件：-f <yaml>")
	}
	b, err := os.ReadFile(*file)
	if err != nil {
		return err
	}

	var errs, warns []string
	for i, line := range strings.Split(string(b), "\n") {
		m := kvRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		key, rawVal := strings.ToLower(m[2]), strings.TrimSpace(m[3])
		val := stripComment(rawVal)
		val = strings.Trim(val, `"'`)
		val = strings.TrimSpace(val)
		if val == "" || strings.HasPrefix(val, config.EncPrefix) {
			continue // 空值（走 env 注入）或已是密文
		}
		lineNo := i + 1
		for _, k := range highRiskKeys {
			if strings.Contains(key, k) {
				errs = append(errs, fmt.Sprintf("  第 %d 行: %s 为明文敏感值（请用 configctl encrypt 加密为 %s 开头）", lineNo, m[2], config.EncPrefix))
				break
			}
		}
		for _, k := range lowRiskKeys {
			if strings.Contains(key, k) {
				warns = append(warns, fmt.Sprintf("  第 %d 行: %s 为明文（用户名敏感度较低，可按需加密）", lineNo, m[2]))
				break
			}
		}
	}

	fmt.Printf("检查 %s：明文敏感值 %d 处，提示 %d 处\n", *file, len(errs), len(warns))
	for _, w := range warns {
		fmt.Println(w)
	}
	if len(errs) > 0 {
		for _, e := range errs {
			fmt.Println(e)
		}
		if *warnOnly {
			fmt.Printf("（--warn-only）%s 存在明文敏感值但不阻断\n", *file)
			return nil
		}
		return fmt.Errorf("存在明文敏感值，禁止推送到 Nacos")
	}
	return nil
}
