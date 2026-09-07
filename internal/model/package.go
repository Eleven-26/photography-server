package model

import "photography-server/internal/enum"

// Package 套餐（版本化管理：被订单引用后改价需生成新版本）
type Package struct {
	TenantBase
	Code             string             `gorm:"column:code;size:20;not null;uniqueIndex:uk_package_code,priority:1;comment:套餐编号 PK-xxx" json:"code"`
	StoreID          int64              `gorm:"column:store_id;index;comment:所属门店ID" json:"store_id"`
	Name             string             `gorm:"column:name;size:100;not null;comment:套餐名称" json:"name"`
	Cover            string             `gorm:"column:cover;size:500;comment:套餐封面图" json:"cover"`
	Category         string             `gorm:"column:category;size:50;comment:套餐类型(婚纱/写真/儿童/全家福/活动跟拍等)" json:"category"`
	SuitableFor      string             `gorm:"column:suitable_for;size:100;comment:适合人群(逗号分隔)" json:"suitable_for"`
	Introduction     string             `gorm:"column:introduction;size:500;comment:套餐简介(一句话)" json:"introduction"`
	BasePrice        float64            `gorm:"column:base_price;type:decimal(12,2);comment:基础套餐价" json:"base_price"`
	DepositRate      float64            `gorm:"column:deposit_rate;type:decimal(5,2);default:30;comment:定金比例(%)" json:"deposit_rate"`
	DepositAmt       float64            `gorm:"column:deposit_amt;type:decimal(12,2);comment:定金金额(基础价×比例)" json:"deposit_amt"`
	PhotosIncluded   int                `gorm:"column:photos_included;comment:包含精修张数" json:"photos_included"`
	RawCount         int                `gorm:"column:raw_count;default:0;comment:原片数量" json:"raw_count"`
	RevisionCount    int                `gorm:"column:revision_count;default:2;comment:修改次数" json:"revision_count"`
	DeliveryDays     int                `gorm:"column:delivery_days;default:7;comment:交付周期(工作日)" json:"delivery_days"`
	DownloadDays     int                `gorm:"column:download_days;default:30;comment:成片下载有效期(天)" json:"download_days"`
	LocationMode     string             `gorm:"column:location_mode;size:20;not null;default:custom;comment:地点模式 fixed-固定 custom-客户可选" json:"location_mode"`
	Locations        string             `gorm:"column:locations;type:text;comment:可选地点(JSON数组)" json:"locations"`
	ReschedulePolicy string             `gorm:"column:reschedule_policy;size:200;comment:改期政策文案" json:"reschedule_policy"`
	CancelPolicy     string             `gorm:"column:cancel_policy;size:200;comment:取消政策文案" json:"cancel_policy"`
	ShootHours       float64            `gorm:"column:shoot_hours;comment:拍摄时长(小时)" json:"shoot_hours"`
	ContentDesc      string             `gorm:"column:content_desc;type:text;comment:包含内容说明" json:"content_desc"`
	AddonUnitPrice   float64            `gorm:"column:addon_unit_price;type:decimal(12,2);default:0;comment:加选精修单价" json:"addon_unit_price"`
	Status           enum.PackageStatus `gorm:"column:status;type:tinyint;default:1;comment:状态 1-草稿 2-已上架 3-已下线" json:"status"`
	Version          int                `gorm:"column:version;default:1;comment:套餐版本号" json:"version"`
	BaseVersion      int                `gorm:"column:base_version;default:0;comment:上一版本号(0表示初始版本)" json:"base_version"`
	PublishedAt      *string            `gorm:"column:published_at;comment:上架时间" json:"published_at"`
}

func (Package) TableName() string { return "biz_package" }
