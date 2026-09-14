package model

import (
	"reflect"
	"strings"
	"testing"
)

// TestEnumColumnsAreInt 回归（2026-09-14 线上故障）：
// biz_upload.file_type 与 biz_delivery_item.file_type / kind 在 DDL 中都是 tinyint，
// 模型一旦声明为 string 并向数据库写入「图片」「sample」这类文本，MySQL 严格模式
// （STRICT_TRANS_TABLES）会直接拒收：
//
//	Incorrect integer value: '\xE5\x9B\xBE\xE7\x89\x87' for column 'file_type' at row 1
//
// 此处锁定这三个字段必须是「整型枚举 + tinyint 列」，防止再次回退为字符串。
func TestEnumColumnsAreInt(t *testing.T) {
	cases := []struct {
		model  interface{}
		field  string
		column string
	}{
		{Upload{}, "FileType", "file_type"},
		{DeliveryItem{}, "FileType", "file_type"},
		{DeliveryItem{}, "Kind", "kind"},
		{SysNotification{}, "Type", "type"},
	}
	for _, c := range cases {
		f, ok := reflect.TypeOf(c.model).FieldByName(c.field)
		if !ok {
			t.Fatalf("%T 缺少字段 %s", c.model, c.field)
		}
		if f.Type.Kind() != reflect.Int {
			t.Errorf("%T.%s 类型 = %s，必须为整型枚举（%s 是 tinyint 列，写字符串会被严格模式拒收）",
				c.model, c.field, f.Type, c.column)
		}
		tag := f.Tag.Get("gorm")
		if !strings.Contains(tag, "column:"+c.column) {
			t.Errorf("%T.%s gorm tag = %q，应映射到列 %s", c.model, c.field, tag, c.column)
		}
		if !strings.Contains(tag, "type:tinyint") {
			t.Errorf("%T.%s gorm tag = %q，应显式声明 type:tinyint", c.model, c.field, tag)
		}
	}
}
