package service

import (
	"path/filepath"
	"strings"
	"testing"

	"photography-server/internal/enum"
)

// TestDetectFileTypeByURL 覆盖交付链路的类型推导：
// file_type 不再由客户端上报，改由服务端按 URL 后缀判定，
// 因此 query/fragment 剥离、大小写、无扩展名、路径穿越都必须稳。
func TestDetectFileTypeByURL(t *testing.T) {
	cases := []struct {
		name string
		url  string
		want enum.UploadType
	}{
		{"jpg", "/uploads/202609/ab12cd34.jpg", enum.UploadTypeImage},
		{"uppercase_ext", "/uploads/202609/ab12cd34.JPG", enum.UploadTypeImage},
		{"webp_with_query", "/uploads/202609/ab12cd34.webp?v=2", enum.UploadTypeImage},
		{"heic", "/uploads/202609/ab12cd34.heic", enum.UploadTypeImage},
		{"mp4", "/uploads/202609/ab12cd34.mp4", enum.UploadTypeVideo},
		{"mov_with_fragment", "/uploads/202609/ab12cd34.mov#t=1", enum.UploadTypeVideo},
		{"full_url", "https://cdn.example.com/uploads/202609/ab12cd34.png", enum.UploadTypeImage},
		{"no_ext", "/uploads/202609/ab12cd34", enum.UploadTypeFile},
		{"empty", "", enum.UploadTypeFile},
		{"traversal", "/uploads/../../etc/passwd", enum.UploadTypeFile},
		{"non_whitelist_ext", "/uploads/202609/ab12cd34.exe", enum.UploadTypeFile},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := detectFileTypeByURL(c.url); got != c.want {
				t.Errorf("detectFileTypeByURL(%q) = %d, want %d", c.url, got, c.want)
			}
		})
	}
}

// TestSanitizeExt 白名单与路径穿越：仅取 Base 后成对白名单的尾缀。
func TestSanitizeExt(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"a.jpg", ".jpg"},
		{"a.JPG", ".jpg"},
		{"../../etc/passwd", ""},
		{"a.tar.gz", ".gz"},
		{"a.", ""},
		{"noext", ""},
		{"", ""},
		{"a.jpg.exe", ".exe"},
		{"a.j-p-g", ""},
		{"a.abcd", ".abcd"},             // 白名单上限内
		{"a.abcdefghij", ".abcdefghij"}, // 11 字符，恰在上限
		{"a.abcdefghijk", ""},           // 12 字符，超上限被拒
	}
	for _, c := range cases {
		if got := sanitizeExt(c.in); got != c.want {
			t.Errorf("sanitizeExt(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// TestUploadTarget 锁定「公开 / 私有」两条上传路径的映射关系。
//
// 落盘子目录与 URL 前缀必须**同步**切换：只改一侧（例如 URL 给 /media 但文件仍落在
// UploadDir 根下）会让请求 404，而上传接口照常返回成功 —— 症状是"上传成功但图片打不开"，
// 极难反查。另一层意图：公开文件必须落在 media/ 子目录里，不能与鉴权目录混放，
// 否则放开 /media 等于顺带暴露同目录下的客户隐私文件。
func TestUploadTarget(t *testing.T) {
	cases := []struct {
		name    string
		public  bool
		month   string
		wantDir string
		wantURL string
	}{
		{"私有：落根目录 + /uploads 前缀", false, "202609", "202609", "/uploads/"},
		{"公开：落 media 子目录 + /media 前缀", true, "202609", filepath.Join("media", "202609"), "/media/"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir, prefix := uploadTarget(c.public, c.month)
			if dir != c.wantDir {
				t.Errorf("落盘目录 = %q, want %q", dir, c.wantDir)
			}
			if prefix != c.wantURL {
				t.Errorf("URL 前缀 = %q, want %q", prefix, c.wantURL)
			}
		})
	}

	// 常量与行为一致性：公开路径必须落在 PublicMediaDir 之下、URL 用 PublicMediaURLPrefix
	dir, prefix := uploadTarget(true, "202609")
	if !strings.HasPrefix(dir, PublicMediaDir+string(filepath.Separator)) {
		t.Errorf("公开文件应落在 %s/ 下，得到 %q", PublicMediaDir, dir)
	}
	if prefix != PublicMediaURLPrefix {
		t.Errorf("公开 URL 前缀应与 PublicMediaURLPrefix 一致，得到 %q", prefix)
	}
	// 私有路径不得出现 media 段（防有人把两条路径写成同一个）
	if privDir, privPrefix := uploadTarget(false, "202609"); strings.Contains(privDir, PublicMediaDir) {
		t.Errorf("私有文件不应落在 %s 下，得到 %q（前缀 %q）", PublicMediaDir, privDir, privPrefix)
	}
}

// TestEnumValuesMatchDDL 锁定枚举值与 DDL tinyint 口径一致（防漂移）。
// 历史故障：模型字段曾声明为 string 并写入中文名，导致
// "Incorrect integer value: '\xE5\x9B\xBE\xE7\x89\x87' for column 'file_type'"。
func TestEnumValuesMatchDDL(t *testing.T) {
	if enum.UploadTypeImage != 1 || enum.UploadTypeVideo != 2 || enum.UploadTypeFile != 3 {
		t.Fatalf("UploadType 与 DDL 不一致: image=%d video=%d file=%d",
			enum.UploadTypeImage, enum.UploadTypeVideo, enum.UploadTypeFile)
	}
	if enum.DeliveryItemKindSample != 1 || enum.DeliveryItemKindSelected != 2 || enum.DeliveryItemKindRetouched != 3 {
		t.Fatalf("DeliveryItemKind 与 DDL 不一致: sample=%d selected=%d retouched=%d",
			enum.DeliveryItemKindSample, enum.DeliveryItemKindSelected, enum.DeliveryItemKindRetouched)
	}
	if got := enum.DeliveryItemKindName(enum.DeliveryItemKindRetouched); got != "精修成品" {
		t.Errorf("DeliveryItemKindName(3) = %q, want 精修成品", got)
	}
}
