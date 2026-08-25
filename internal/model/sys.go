package model

// SysCompany 公司/工作室
type SysCompany struct {
	Base
	Name         string `gorm:"column:name;size:100;not null;comment:公司/工作室名称" json:"name"`
	Logo         string `gorm:"column:logo;size:500;comment:LOGO地址" json:"logo"`
	ContactName  string `gorm:"column:contact_name;size:50;comment:联系人" json:"contact_name"`
	ContactPhone string `gorm:"column:contact_phone;size:20;comment:联系电话" json:"contact_phone"`
	Address      string `gorm:"column:address;size:200;comment:地址" json:"address"`
	Status       int    `gorm:"column:status;type:tinyint;default:1;comment:状态 1-正常 0-停用" json:"status"`
}

func (SysCompany) TableName() string { return "sys_company" }

// SysStore 门店（公司下多门店）
type SysStore struct {
	TenantBase
	Name    string `gorm:"column:name;size:100;not null;comment:门店名称" json:"name"`
	Address string `gorm:"column:address;size:200;comment:门店地址" json:"address"`
	Phone   string `gorm:"column:phone;size:20;comment:门店电话" json:"phone"`
	Status  int    `gorm:"column:status;type:tinyint;default:1;comment:状态 1-正常 0-停用" json:"status"`
}

func (SysStore) TableName() string { return "sys_store" }

// SysRole 角色
type SysRole struct {
	TenantBase
	Name   string `gorm:"column:name;size:50;not null;comment:角色名称" json:"name"`
	Code   string `gorm:"column:code;size:50;not null;comment:角色编码 admin-超级管理员 manager-店长 photograher-摄影师 sales-销售" json:"code"`
	Remark string `gorm:"column:remark;size:200;comment:备注" json:"remark"`
	Status int    `gorm:"column:status;type:tinyint;default:1;comment:状态 1-启用 0-停用" json:"status"`
}

func (SysRole) TableName() string { return "sys_role" }

// SysUser 后台管理员/员工
type SysUser struct {
	TenantBase
	StoreID     int64   `gorm:"column:store_id;index;comment:所属门店ID" json:"store_id"`
	Username    string  `gorm:"column:username;size:50;not null;comment:登录账号" json:"username"`
	Password    string  `gorm:"column:password;size:255;not null;comment:登录密码(bcrypt)" json:"-"`
	Nickname    string  `gorm:"column:nickname;size:50;comment:姓名/昵称" json:"nickname"`
	Mobile      string  `gorm:"column:mobile;size:20;comment:手机号" json:"mobile"`
	Email       string  `gorm:"column:email;size:100;comment:邮箱" json:"email"`
	Avatar      string  `gorm:"column:avatar;size:500;comment:头像地址" json:"avatar"`
	RoleID      int64   `gorm:"column:role_id;index;comment:角色ID" json:"role_id"`
	Status      int     `gorm:"column:status;type:tinyint;default:1;comment:状态 1-启用 0-停用" json:"status"`
	LastLoginAt *string `gorm:"column:last_login_at;comment:最近登录时间" json:"last_login_at"`
	LastLoginIP string  `gorm:"column:last_login_ip;size:50;comment:最近登录IP" json:"last_login_ip"`
}

func (SysUser) TableName() string { return "sys_user" }

// SysOperationLog 操作日志
type SysOperationLog struct {
	TenantBase
	UserID   int64  `gorm:"column:user_id;index;comment:操作人ID" json:"user_id"`
	Username string `gorm:"column:username;size:50;comment:操作人账号" json:"username"`
	Module   string `gorm:"column:module;size:50;comment:模块" json:"module"`
	Action   string `gorm:"column:action;size:100;comment:操作行为" json:"action"`
	Method   string `gorm:"column:method;size:10;comment:请求方法" json:"method"`
	Path     string `gorm:"column:path;size:200;comment:请求路径" json:"path"`
	Params   string `gorm:"column:params;type:text;comment:请求参数" json:"params"`
	IP       string `gorm:"column:ip;size:50;comment:请求IP" json:"ip"`
	Status   int    `gorm:"column:status;type:tinyint;comment:状态 1-成功 0-失败" json:"status"`
	Duration int64  `gorm:"column:duration;comment:耗时(毫秒)" json:"duration"`
}

func (SysOperationLog) TableName() string { return "sys_operation_log" }

// SysNotification 站内通知
type SysNotification struct {
	TenantBase
	ReceiverID int64   `gorm:"column:receiver_id;index;comment:接收人ID" json:"receiver_id"`
	Type       string  `gorm:"column:type;size:20;comment:类型 order-订单 finance-财务 system-系统" json:"type"`
	Title      string  `gorm:"column:title;size:100;comment:标题" json:"title"`
	Content    string  `gorm:"column:content;size:500;comment:内容" json:"content"`
	BizType    string  `gorm:"column:biz_type;size:20;comment:业务类型 order-订单 refund-退款" json:"biz_type"`
	BizID      int64   `gorm:"column:biz_id;comment:业务ID" json:"biz_id"`
	IsRead     int     `gorm:"column:is_read;type:tinyint;default:0;comment:是否已读 0-未读 1-已读" json:"is_read"`
	ReadAt     *string `gorm:"column:read_at;comment:已读时间" json:"read_at"`
}

func (SysNotification) TableName() string { return "sys_notification" }
