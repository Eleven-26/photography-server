package model

import "photography-server/internal/enum"

// Delivery 交付单（选片精修交付流程）
type Delivery struct {
	TenantBase
	Code                string             `gorm:"column:code;size:20;not null;uniqueIndex:uk_delivery_code,priority:1;comment:交付单编号 DV-xxx" json:"code"`
	OrderID             int64              `gorm:"column:order_id;index;comment:订单ID" json:"order_id"`
	CustomerID          int64              `gorm:"column:customer_id;index;comment:客户ID" json:"customer_id"`
	CustomerName        string             `gorm:"column:customer_name;size:50;comment:客户姓名(快照)" json:"customer_name"`
	Stage               enum.DeliveryStage `gorm:"column:stage;type:tinyint;default:1;comment:阶段 1-待上传样片 2-客户选片中 3-精修进行中 4-待确认交付 5-已交付" json:"stage"`
	RawCount            int                `gorm:"column:raw_count;default:0;comment:原片数量(计划)" json:"raw_count"`
	RetouchTarget       int                `gorm:"column:retouch_target;default:0;comment:计划精修张数" json:"retouch_target"`
	SampleCount         int                `gorm:"column:sample_count;default:0;comment:样片数量" json:"sample_count"`
	SelectedCount       int                `gorm:"column:selected_count;default:0;comment:客户已选张数" json:"selected_count"`
	SelectDeadline      *string            `gorm:"column:select_deadline;comment:选片截止时间(逾期默认全选)" json:"select_deadline"`
	ExtraSelectedCount  int                `gorm:"column:extra_selected_count;default:0;comment:加选张数(超出套餐精修数)" json:"extra_selected_count"`
	ExtraFee            float64            `gorm:"column:extra_fee;type:decimal(12,2);default:0;comment:加选费用(计入尾款)" json:"extra_fee"`
	ExtraConfirmed      int                `gorm:"column:extra_confirmed;type:tinyint;default:0;comment:客户是否确认加片 0-待确认 1-已确认" json:"extra_confirmed"`
	RetouchVersion      int                `gorm:"column:retouch_version;default:1;comment:精修轮次(最终成片 V1/V2)" json:"retouch_version"`
	SentFinalAt         *string            `gorm:"column:sent_final_at;comment:发送最终确认时间" json:"sent_final_at"`
	CustomerConfirmedAt *string            `gorm:"column:customer_confirmed_at;comment:客户确认成片时间" json:"customer_confirmed_at"`
	RetouchedCount      int                `gorm:"column:retouched_count;default:0;comment:精修完成张数" json:"retouched_count"`
	SelectedAt          *string            `gorm:"column:selected_at;comment:选片完成时间" json:"selected_at"`
	DeliveredAt         *string            `gorm:"column:delivered_at;comment:交付时间" json:"delivered_at"`
	Remark              string             `gorm:"column:remark;size:500;comment:备注" json:"remark"`
	OperatorID          int64              `gorm:"column:operator_id;comment:当前处理人ID" json:"operator_id"`
}

func (Delivery) TableName() string { return "biz_delivery" }

// DeliveryItem 交付明细（样片/精修文件）
type DeliveryItem struct {
	TenantBase
	DeliveryID       int64   `gorm:"column:delivery_id;index;comment:交付单ID" json:"delivery_id"`
	OrderID          int64   `gorm:"column:order_id;index;comment:订单ID" json:"order_id"`
	URL              string  `gorm:"column:url;size:500;not null;comment:文件地址" json:"url"`
	FileType         string  `gorm:"column:file_type;size:10;comment:类型 image-图片 video-视频" json:"file_type"`
	Kind             string  `gorm:"column:kind;size:20;comment:用途 sample-样片 selected-已选 retouched-精修成品" json:"kind"`
	Filename         string  `gorm:"column:filename;size:200;comment:原始文件名" json:"filename"`
	Size             int64   `gorm:"column:size;comment:文件大小(字节)" json:"size"`
	IsSelected       int     `gorm:"column:is_selected;type:tinyint;default:0;comment:客户是否选中 0-否 1-是" json:"is_selected"`
	FeedbackContent  string  `gorm:"column:feedback_content;size:500;comment:客户修图反馈内容" json:"feedback_content"`
	FeedbackTypes    string  `gorm:"column:feedback_types;size:100;comment:反馈修改类型(逗号分隔)" json:"feedback_types"`
	FeedbackPriority string  `gorm:"column:feedback_priority;size:10;comment:反馈优先级 normal-important-urgent" json:"feedback_priority"`
	FeedbackStatus   int     `gorm:"column:feedback_status;type:tinyint;default:0;comment:反馈状态 0-无 1-待处理 2-已处理" json:"feedback_status"`
	HandledAt        *string `gorm:"column:handled_at;comment:反馈处理时间" json:"handled_at"`
	HandleRemark     string  `gorm:"column:handle_remark;size:200;comment:处理备注(如确认加项)" json:"handle_remark"`
}

func (DeliveryItem) TableName() string { return "biz_delivery_item" }
