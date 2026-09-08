package config

import (
	"encoding/base64"
	"os"
	"strings"
	"testing"
)

const testKeyB64 = "MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY=" // 32 字节

func setTestKey(t *testing.T) *Cipher {
	t.Helper()
	t.Setenv(secretEnv, testKeyB64)
	c, err := LoadCipher()
	if err != nil {
		t.Fatalf("LoadCipher: %v", err)
	}
	if !c.Enabled() {
		t.Fatal("密钥未启用")
	}
	return c
}

func TestCipherRoundTrip(t *testing.T) {
	c := setTestKey(t)
	for _, plain := range []string{"", "p@ss:word#1", "中文密码", strings.Repeat("x", 512)} {
		enc, err := c.Encrypt(plain)
		if err != nil {
			t.Fatalf("Encrypt: %v", err)
		}
		if !strings.HasPrefix(enc, EncPrefix) {
			t.Fatalf("密文缺少前缀: %s", enc)
		}
		if enc == EncPrefix+plain {
			t.Fatal("未产生密文")
		}
		got, err := c.Decrypt(enc)
		if err != nil {
			t.Fatalf("Decrypt: %v", err)
		}
		if got != plain {
			t.Fatalf("解密结果不符: %q != %q", got, plain)
		}
	}
}

func TestCipherNonCiphertextPassthrough(t *testing.T) {
	c := setTestKey(t)
	if got, err := c.Decrypt("plain-value"); err != nil || got != "plain-value" {
		t.Fatalf("非密文应原样返回: %q %v", got, err)
	}
}

func TestCipherTamperedOrWrongKey(t *testing.T) {
	c := setTestKey(t)
	enc, _ := c.Encrypt("secret")
	raw, _ := strings.CutPrefix(enc, EncPrefix)
	blob, _ := base64.StdEncoding.DecodeString(raw)
	blob[len(blob)-1] ^= 0xFF // 篡改尾部的 GCM tag
	bad := EncPrefix + base64.StdEncoding.EncodeToString(blob)
	if _, err := c.Decrypt(bad); err == nil {
		t.Fatal("被篡改的密文应当解密失败")
	}

	t.Setenv(secretEnv, base64.StdEncoding.EncodeToString(make([]byte, 32)))
	other, err := LoadCipher()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := other.Decrypt(enc); err == nil {
		t.Fatal("错误的密钥应当解密失败")
	}
}

func TestCipherMissingKey(t *testing.T) {
	t.Setenv(secretEnv, "")
	t.Setenv(secretFileEnv, "")
	c, err := LoadCipher()
	if err != nil {
		t.Fatal(err)
	}
	if c.Enabled() {
		t.Fatal("未配置密钥时 Enabled 应为 false")
	}
	if _, err := c.Encrypt("x"); err == nil {
		t.Fatal("缺密钥时加密应报错")
	}
	if _, err := c.Decrypt(EncPrefix + base64.StdEncoding.EncodeToString(make([]byte, 40))); err == nil {
		t.Fatal("缺密钥时解密 ENC 密文应报错")
	}
	// 缺密钥但配置无密文 → DecryptSecrets 应静默跳过
	type inner struct{ Secret string }
	type outer struct {
		Password string
		Nested   inner
		List     []string
	}
	o := &outer{Password: "plain", Nested: inner{Secret: "plain2"}, List: []string{"a", "b"}}
	if err := DecryptSecrets(c, o); err != nil {
		t.Fatalf("无密钥时应跳过: %v", err)
	}
}

func TestCipherKeyFromFile(t *testing.T) {
	dir := t.TempDir()
	p := dir + "/kek"
	if err := os.WriteFile(p, []byte(testKeyB64+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(secretEnv, "")
	t.Setenv(secretFileEnv, p)
	c, err := LoadCipher()
	if err != nil {
		t.Fatal(err)
	}
	if !c.Enabled() {
		t.Fatal("文件密钥未生效")
	}
	if _, err := c.Encrypt("x"); err != nil {
		t.Fatal(err)
	}
}

func TestDecryptSecretsRecursive(t *testing.T) {
	c := setTestKey(t)
	pwd, _ := c.Encrypt("db-pass")
	tok, _ := c.Encrypt("xxl-token")

	type es struct {
		Username string
		Password string
		Urls     []string
	}
	cfg := struct {
		DB     struct{ Password string }
		ES     es
		XxlJob struct{ AccessToken string }
		Port   int
	}{}
	cfg.DB.Password = pwd
	cfg.ES = es{Username: "elastic", Password: pwd, Urls: []string{"http://a:9200", tok}}
	cfg.XxlJob.AccessToken = tok
	cfg.Port = 8080

	if err := DecryptSecrets(c, &cfg); err != nil {
		t.Fatalf("DecryptSecrets: %v", err)
	}
	if cfg.DB.Password != "db-pass" || cfg.ES.Password != "db-pass" || cfg.XxlJob.AccessToken != "xxl-token" {
		t.Fatalf("嵌套字段未解密: %+v", cfg)
	}
	if cfg.ES.Urls[1] != "xxl-token" {
		t.Fatalf("切片元素未解密: %v", cfg.ES.Urls)
	}
	if cfg.ES.Username != "elastic" || cfg.Port != 8080 {
		t.Fatalf("非密文字段被改动: %+v", cfg)
	}
}

func TestDecryptSecretsBadCipher(t *testing.T) {
	c := setTestKey(t)
	cfg := struct{ Password string }{Password: EncPrefix + "!!!not-base64!!!"}
	err := DecryptSecrets(c, &cfg)
	if err == nil {
		t.Fatal("坏密文应报错")
	}
	if !strings.Contains(err.Error(), "Password") {
		t.Fatalf("错误信息应指出字段名: %v", err)
	}
}
