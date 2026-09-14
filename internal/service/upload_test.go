package service

import (
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
