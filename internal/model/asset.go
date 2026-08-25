package model

// Asset 作品集
type Asset struct {
	TenantBase
	Code         string  `gorm:"column:code;size:20;not null;uniqueIndex:uk_asset_code,priority:1;comment:作品编号 WK-xxx" json:"code"`
	Title        string  `gorm:"column:title;size:100;comment:作品标题" json:"title"`
	Category     string  `gorm:"column:category;size:50;comment:作品类型(婚纱/写真/儿童/全家福/活动跟拍等)" json:"category"`
	Cover        string  `gorm:"column:cover;size:500;comment:封面图" json:"cover"`
	Images       string  `gorm:"column:images;type:text;comment:作品图片(逗号分隔)" json:"images"`
	Description  string  `gorm:"column:description;size:1000;comment:作品描述" json:"description"`
	Photographer string  `gorm:"column:photographer;size:50;comment:摄影师" json:"photographer"`
	Model        string  `gorm:"column:model;size:50;comment:模特" json:"model"`
	Location     string  `gorm:"column:location;size:100;comment:拍摄地点" json:"location"`
	Status       string  `gorm:"column:status;size:20;default:draft;comment:状态 draft-草稿 published-已发布" json:"status"`
	ViewCount    int64   `gorm:"column:view_count;default:0;comment:浏览数" json:"view_count"`
	PublishedAt  *string `gorm:"column:published_at;comment:发布时间" json:"published_at"`
}

func (Asset) TableName() string { return "biz_asset" }

// CalendarBlock 档期锁定
type CalendarBlock struct {
	TenantBase
	StoreID        int64  `gorm:"column:store_id;index;comment:所属门店ID" json:"store_id"`
	OrderID        int64  `gorm:"column:order_id;index;comment:关联订单ID" json:"order_id"`
	CustomerID     int64  `gorm:"column:customer_id;index;comment:关联客户ID" json:"customer_id"`
	CustomerName   string `gorm:"column:customer_name;size:50;comment:客户姓名(快照)" json:"customer_name"`
	Date           string `gorm:"column:date;size:10;index;comment:拍摄日期 yyyy-MM-dd" json:"date"`
	TimeRange      string `gorm:"column:time_range;size:50;comment:时间段" json:"time_range"`
	ProjectType    string `gorm:"column:project_type;size:50;comment:项目类型" json:"project_type"`
	PhotographerID int64  `gorm:"column:photographer_id;index;comment:摄影师ID" json:"photographer_id"`
	Photographer   string `gorm:"column:photographer;size:50;comment:摄影师姓名" json:"photographer"`
	Status         string `gorm:"column:status;size:20;default:locked;comment:状态 locked-已锁定 cancelled-已取消" json:"status"`
	Remark         string `gorm:"column:remark;size:200;comment:备注" json:"remark"`
}

func (CalendarBlock) TableName() string { return "biz_calendar_block" }
