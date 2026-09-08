package config

// 配置敏感值加解密（方案 B：字段级 ENC 密文）
//
// 设计要点：
//  1. 密文格式 ENCv1:<base64(nonce|ciphertext|tag)>，算法 AES-256-GCM（自带完整性校验）
//  2. 主密钥（KEK）来自 secret 文件或环境变量，**不进 Nacos、不进 git**；
//     Nacos 里只有密文，拖库也无法还原
//  3. 未配置密钥时不启用解密（dev 明文可用）；一旦遇到 ENCv1 密文则 fail-fast
//  4. previous 密钥用于轮换期并存（只参与解密，加密恒用 current）
//
// 密钥优先级：APP_CONFIG_SECRET_FILE（文件，Docker/K8s secret）> APP_CONFIG_SECRET（环境变量）

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
)

const (
	// EncPrefix 密文前缀
	EncPrefix = "ENCv1:"

	secretFileEnv = "APP_CONFIG_SECRET_FILE" // 主密钥文件（推荐）
	secretEnv     = "APP_CONFIG_SECRET"      // 主密钥环境变量（base64/hex/32 字节原文）
	prevSecretEnv = "APP_CONFIG_SECRET_PREV" // 轮换期旧密钥（只解密）

	keyLen   = 32 // AES-256
	nonceLen = 12 // GCM 标准 nonce
)

// Cipher 配置加解密器；未配置密钥时 Enabled() 为 false
type Cipher struct {
	current  []byte
	previous []byte
}

// LoadCipher 按优先级加载主密钥；两者都缺失时返回未启用的 Cipher（不报错）
func LoadCipher() (*Cipher, error) {
	c := &Cipher{}
	var err error
	if c.current, err = loadKey(secretEnv, secretFileEnv); err != nil {
		return nil, err
	}
	if c.previous, err = loadKey(prevSecretEnv, ""); err != nil {
		return nil, err
	}
	return c, nil
}

func loadKey(envName, fileEnvName string) ([]byte, error) {
	raw := ""
	if fileEnvName != "" {
		if p := os.Getenv(fileEnvName); p != "" {
			b, err := os.ReadFile(p)
			if err != nil {
				return nil, fmt.Errorf("读取主密钥文件失败（%s=%s）: %w", fileEnvName, p, err)
			}
			raw = strings.TrimSpace(string(b))
		}
	}
	if raw == "" {
		raw = strings.TrimSpace(os.Getenv(envName))
	}
	if raw == "" {
		return nil, nil
	}
	key, err := parseKey(raw)
	if err != nil {
		return nil, fmt.Errorf("解析主密钥失败（%s）: %w", envName, err)
	}
	return key, nil
}

// parseKey 支持 base64（44 字符）/ hex（64 字符）/ 32 字节原文
func parseKey(s string) ([]byte, error) {
	if b, err := base64.StdEncoding.DecodeString(s); err == nil && len(b) == keyLen {
		return b, nil
	}
	if b, err := hex.DecodeString(s); err == nil && len(b) == keyLen {
		return b, nil
	}
	if len(s) == keyLen {
		return []byte(s), nil
	}
	return nil, errors.New("密钥需为 32 字节（base64 44 字符 / hex 64 字符 / 原文 32 字符）")
}

// Enabled 是否已配置主密钥
func (c *Cipher) Enabled() bool { return c != nil && len(c.current) == keyLen }

// Encrypt 加密明文，输出 ENCv1: 密文
func (c *Cipher) Encrypt(plain string) (string, error) {
	if !c.Enabled() {
		return "", errors.New("未配置主密钥（APP_CONFIG_SECRET / APP_CONFIG_SECRET_FILE），无法加密")
	}
	block, err := aes.NewCipher(c.current)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, nonceLen)
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("生成 nonce 失败: %w", err)
	}
	ct := gcm.Seal(nil, nonce, []byte(plain), nil)
	buf := make([]byte, 0, nonceLen+len(ct))
	buf = append(buf, nonce...)
	buf = append(buf, ct...)
	return EncPrefix + base64.StdEncoding.EncodeToString(buf), nil
}

// Decrypt 解密 ENCv1 密文；非密文原样返回
func (c *Cipher) Decrypt(s string) (string, error) {
	raw, ok := strings.CutPrefix(s, EncPrefix)
	if !ok {
		return s, nil
	}
	if !c.Enabled() {
		return "", errors.New("未配置主密钥（APP_CONFIG_SECRET / APP_CONFIG_SECRET_FILE），无法解密 ENCv1 密文")
	}
	blob, err := base64.StdEncoding.DecodeString(strings.TrimSpace(raw))
	if err != nil {
		return "", errors.New("密文 base64 格式非法")
	}
	if len(blob) < nonceLen+16 {
		return "", errors.New("密文长度不足")
	}
	var lastErr error
	for _, k := range [][]byte{c.current, c.previous} {
		if len(k) != keyLen {
			continue
		}
		out, err := gcmOpen(k, blob)
		if err == nil {
			return out, nil
		}
		lastErr = err
	}
	return "", fmt.Errorf("解密失败（主密钥不匹配或密文被篡改）: %v", lastErr)
}

func gcmOpen(key, blob []byte) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	plain, err := gcm.Open(nil, blob[:nonceLen], blob[nonceLen:], nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

// DecryptSecrets 递归解密 target 中所有 ENCv1 字符串字段（就地修改，明文只存在于内存）
func DecryptSecrets(c *Cipher, target any) error {
	if c == nil || !c.Enabled() {
		return nil
	}
	v := reflect.ValueOf(target)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return errors.New("DecryptSecrets: 需要非 nil 指针")
	}
	return decryptValue(c, v.Elem(), "")
}

func decryptValue(c *Cipher, v reflect.Value, path string) error {
	switch v.Kind() {
	case reflect.String:
		if !v.CanSet() || !strings.HasPrefix(v.String(), EncPrefix) {
			return nil
		}
		plain, err := c.Decrypt(v.String())
		if err != nil {
			return fmt.Errorf("解密配置失败（字段 %s）: %w", strings.TrimPrefix(path, "."), err)
		}
		v.SetString(plain)
	case reflect.Struct:
		t := v.Type()
		for i := 0; i < v.NumField(); i++ {
			if t.Field(i).PkgPath != "" { // 未导出字段跳过
				continue
			}
			if err := decryptValue(c, v.Field(i), path+"."+t.Field(i).Name); err != nil {
				return err
			}
		}
	case reflect.Ptr:
		if !v.IsNil() {
			return decryptValue(c, v.Elem(), path)
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			if err := decryptValue(c, v.Index(i), fmt.Sprintf("%s[%d]", path, i)); err != nil {
				return err
			}
		}
	}
	return nil
}
