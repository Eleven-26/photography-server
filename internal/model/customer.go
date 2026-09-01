package model

import "photography-server/internal/enum"

// Customer 客户
type Customer struct {
	TenantBase
	Code        string              `gorm:"column:code;size:20;not null;uniqueIndex:uk_customer_code,priority:1;comment:客户编号 CU-xxx" json:"code"`
	StoreID     int64               `gorm:"column:store_id;index;comment:所属门店ID" json:"store_id"`
	Name        string              `gorm:"column:name;size:50;not null;comment:客户姓名" json:"name"`
	Mobile      string              `gorm:"column:mobile;size:20;comment:手机号" json:"mobile"`
	Wechat      string              `gorm:"column:wechat;size:50;comment:微信号" json:"wechat"`
	Gender      string              `gorm:"column:gender;size:10;comment:性别 male-男 female-女 unknown-未知" json:"gender"`
	Birthday    *string             `gorm:"column:birthday;comment:生日" json:"birthday"`
	Level       enum.CustomerLevel  `gorm:"column:level;type:tinyint;default:1;comment:客户等级 1-普通 2-黄金 3-铂金 4-钻石" json:"level"`
	Source      string              `gorm:"column:source;size:50;comment:客户来源" json:"source"`
	Tags        string              `gorm:"column:tags;size:200;comment:标签(逗号分隔)" json:"tags"`
	Status      enum.CustomerStatus `gorm:"column:status;type:tinyint;default:1;comment:状态 1-潜在 2-活跃 3-流失" json:"status"`
	Remark      string              `gorm:"column:remark;size:500;comment:备注" json:"remark"`
	Avatar      string              `gorm:"column:avatar;size:500;comment:头像地址" json:"avatar"`
	OrderCount  int64               `gorm:"column:order_count;default:0;comment:订单数(冗余)" json:"order_count"`
	TotalAmount float64             `gorm:"column:total_amount;type:decimal(12,2);default:0;comment:累计消费(冗余)" json:"total_amount"`
}

func (Customer) TableName() string { return "crm_customer" }
