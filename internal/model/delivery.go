package model

import "photography-server/internal/enum"

// Delivery 交付单（选片精修交付流程）
type Delivery struct {
	TenantBase
	Code           string             `gorm:"column:code;size:20;not null;uniqueIndex:uk_delivery_code,priority:1;comment:交付单编号 DV-xxx" json:"code"`
	OrderID        int64              `gorm:"column:order_id;index;comment:订单ID" json:"order_id"`
	CustomerID     int64              `gorm:"column:customer_id;index;comment:客户ID" json:"customer_id"`
	CustomerName   string             `gorm:"column:customer_name;size:50;comment:客户姓名(快照)" json:"customer_name"`
	Stage          enum.DeliveryStage `gorm:"column:stage;type:tinyint;default:1;comment:阶段 1-待上传样片 2-客户选片中 3-精修进行中 4-待确认交付 5-已交付" json:"stage"`
	SampleCount    int                `gorm:"column:sample_count;default:0;comment:样片数量" json:"sample_count"`
	SelectedCount  int                `gorm:"column:selected_count;default:0;comment:客户已选张数" json:"selected_count"`
	RetouchedCount int                `gorm:"column:retouched_count;default:0;comment:精修完成张数" json:"retouched_count"`
	SelectedAt     *string            `gorm:"column:selected_at;comment:选片完成时间" json:"selected_at"`
	DeliveredAt    *string            `gorm:"column:delivered_at;comment:交付时间" json:"delivered_at"`
	Remark         string             `gorm:"column:remark;size:500;comment:备注" json:"remark"`
	OperatorID     int64              `gorm:"column:operator_id;comment:当前处理人ID" json:"operator_id"`
}

func (Delivery) TableName() string { return "biz_delivery" }

// DeliveryItem 交付明细（样片/精修文件）
type DeliveryItem struct {
	TenantBase
	DeliveryID int64  `gorm:"column:delivery_id;index;comment:交付单ID" json:"delivery_id"`
	OrderID    int64  `gorm:"column:order_id;index;comment:订单ID" json:"order_id"`
	URL        string `gorm:"column:url;size:500;not null;comment:文件地址" json:"url"`
	FileType   string `gorm:"column:file_type;size:10;comment:类型 image-图片 video-视频" json:"file_type"`
	Kind       string `gorm:"column:kind;size:20;comment:用途 sample-样片 selected-已选 retouched-精修成品" json:"kind"`
	Filename   string `gorm:"column:filename;size:200;comment:原始文件名" json:"filename"`
	Size       int64  `gorm:"column:size;comment:文件大小(字节)" json:"size"`
}

func (DeliveryItem) TableName() string { return "biz_delivery_item" }
