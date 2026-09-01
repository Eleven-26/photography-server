package model

import "photography-server/internal/enum"

// Lead 线索
type Lead struct {
	TenantBase
	Code         string          `gorm:"column:code;size:20;not null;uniqueIndex:uk_lead_code,priority:1;comment:线索编号 LD-xxx" json:"code"`
	StoreID      int64           `gorm:"column:store_id;index;comment:所属门店ID" json:"store_id"`
	CustomerID   int64           `gorm:"column:customer_id;index;comment:关联客户ID" json:"customer_id"`
	Name         string          `gorm:"column:name;size:50;comment:客户姓名" json:"name"`
	Mobile       string          `gorm:"column:mobile;size:20;comment:手机号" json:"mobile"`
	Source       string          `gorm:"column:source;size:50;comment:线索来源" json:"source"`
	ProjectType  string          `gorm:"column:project_type;size:50;comment:意向项目类型(婚纱/写真/儿童/全家福/活动跟拍等)" json:"project_type"`
	BudgetMin    float64         `gorm:"column:budget_min;type:decimal(12,2);comment:预算区间-下限" json:"budget_min"`
	BudgetMax    float64         `gorm:"column:budget_max;type:decimal(12,2);comment:预算区间-上限" json:"budget_max"`
	Status       enum.LeadStatus `gorm:"column:status;type:tinyint;default:1;comment:状态 1-待回复 2-待报价 3-已报价 4-已成交 5-已流失" json:"status"`
	ShootDate    *string         `gorm:"column:shoot_date;comment:意向拍摄日期" json:"shoot_date"`
	Remark       string          `gorm:"column:remark;size:500;comment:备注" json:"remark"`
	OwnerID      int64           `gorm:"column:owner_id;index;comment:负责人ID" json:"owner_id"`
	NextFollowAt *string         `gorm:"column:next_follow_at;comment:下次跟进时间" json:"next_follow_at"`
	Follower     int64           `gorm:"column:follower;default:0;comment:跟进次数" json:"follower"`
	LastFollowAt *string         `gorm:"column:last_follow_at;comment:最近跟进时间" json:"last_follow_at"`
}

func (Lead) TableName() string { return "crm_lead" }

// Quote 报价单
type Quote struct {
	TenantBase
	Code        string           `gorm:"column:code;size:20;not null;uniqueIndex:uk_quote_code,priority:1;comment:报价单编号 QT-xxx" json:"code"`
	LeadID      int64            `gorm:"column:lead_id;index;comment:关联线索ID" json:"lead_id"`
	CustomerID  int64            `gorm:"column:customer_id;index;comment:关联客户ID" json:"customer_id"`
	PackageID   int64            `gorm:"column:package_id;index;comment:关联套餐ID" json:"package_id"`
	Version     int              `gorm:"column:version;default:1;comment:套餐版本号" json:"version"`
	Title       string           `gorm:"column:title;size:100;comment:报价标题" json:"title"`
	PackageName string           `gorm:"column:package_name;size:100;comment:套餐名称(下单时快照)" json:"package_name"`
	BasePrice   float64          `gorm:"column:base_price;type:decimal(12,2);comment:基础套餐价(快照)" json:"base_price"`
	AddonPrice  float64          `gorm:"column:addon_price;type:decimal(12,2);default:0;comment:加选金额(快照)" json:"addon_price"`
	TotalPrice  float64          `gorm:"column:total_price;type:decimal(12,2);comment:报价总额" json:"total_price"`
	Status      enum.QuoteStatus `gorm:"column:status;type:tinyint;default:1;comment:状态 1-草稿 2-已发送 3-已接受 4-已拒绝 5-已成交" json:"status"`
	Remark      string           `gorm:"column:remark;size:500;comment:备注" json:"remark"`
	OwnerID     int64            `gorm:"column:owner_id;index;comment:负责人ID" json:"owner_id"`
	ShootDate   *string          `gorm:"column:shoot_date;comment:意向拍摄日期" json:"shoot_date"`
}

func (Quote) TableName() string { return "biz_quote" }
