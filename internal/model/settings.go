package model

// PaymentMethod 收款方式
type PaymentMethod struct {
	TenantBase
	Name        string `gorm:"column:name;size:50;not null;comment:收款方式名称" json:"name"`
	Type        string `gorm:"column:type;size:20;comment:类型 wechat-微信 alipay-支付宝 bank-银行转账 cash-现金 other-其他" json:"type"`
	AccountName string `gorm:"column:account_name;size:100;comment:收款账户名称" json:"account_name"`
	AccountNo   string `gorm:"column:account_no;size:100;comment:收款账号" json:"account_no"`
	Qrcode      string `gorm:"column:qrcode;size:500;comment:收款二维码" json:"qrcode"`
	Status      int    `gorm:"column:status;type:tinyint;default:1;comment:状态 1-启用 0-停用" json:"status"`
	Sort        int    `gorm:"column:sort;default:0;comment:排序" json:"sort"`
}

func (PaymentMethod) TableName() string { return "biz_payment_method" }

// Upload 上传文件记录
type Upload struct {
	TenantBase
	StoreID  int64  `gorm:"column:store_id;index;comment:所属门店ID" json:"store_id"`
	BizType  string `gorm:"column:biz_type;size:50;comment:业务类型(订单/交付/作品等)" json:"biz_type"`
	BizID    int64  `gorm:"column:biz_id;comment:业务ID" json:"biz_id"`
	FileType string `gorm:"column:file_type;size:10;comment:文件类型 image-图片 video-视频 file-文件" json:"file_type"`
	FileName string `gorm:"column:file_name;size:200;comment:原始文件名" json:"file_name"`
	FileURL  string `gorm:"column:file_url;size:500;not null;comment:文件访问地址" json:"file_url"`
	FilePath string `gorm:"column:file_path;size:500;comment:服务器存储路径" json:"file_path"`
	Size     int64  `gorm:"column:size;comment:文件大小(字节)" json:"size"`
	UploadBy int64  `gorm:"column:upload_by;comment:上传人ID" json:"upload_by"`
}

func (Upload) TableName() string { return "biz_upload" }
