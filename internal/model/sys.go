package model

import (
	"time"

	"photography-server/internal/enum"
)

// SysCompany 公司/工作室
type SysCompany struct {
	Base
	Name         string `gorm:"column:name;size:100;not null;comment:公司/工作室名称" json:"name"`
	Logo         string `gorm:"column:logo;size:500;comment:LOGO地址" json:"logo"`
	City         string `gorm:"column:city;size:50;comment:所在城市" json:"city"`
	Intro        string `gorm:"column:intro;size:1000;comment:工作室简介" json:"intro"`
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
	Name      string `gorm:"column:name;size:50;not null;comment:角色名称" json:"name"`
	Code      string `gorm:"column:code;size:50;not null;comment:角色编码 admin-超级管理员 manager-店长 photographer-摄影师 sales-销售" json:"code"`
	Remark    string `gorm:"column:remark;size:200;comment:备注" json:"remark"`
	Status    int    `gorm:"column:status;type:tinyint;default:1;comment:状态 1-启用 0-停用" json:"status"`
	DataScope int    `gorm:"column:data_scope;type:tinyint;default:1;comment:数据范围 1-全部数据 2-本门店 3-仅本人" json:"data_scope"`

	// PermissionCount 已配置的权限点数量。非数据库列（gorm:"-"），
	// 由 ListRoles 聚合 sys_role_permission 后填充，供角色列表展示。
	PermissionCount int `gorm:"-" json:"permission_count"`
}

func (SysRole) TableName() string { return "sys_role" }

// SysRolePermission 角色权限关联。
//
// 本表**不继承 TenantBase**——它是纯关联表，无业务语义，且采用「全量覆盖式写入」
// （保存时物理删除该角色旧记录再批量插入，见 repository.ReplaceRolePerms）。
// 若继承 Base 的软删除字段，uk_role_perm(company_id,role_id,permission) 会在
// "删掉某权限再加回来"时命中旧记录而冲突。
type SysRolePermission struct {
	ID         int64     `gorm:"column:id;primaryKey;autoIncrement;comment:主键ID" json:"id"`
	CompanyID  int64     `gorm:"column:company_id;index;comment:公司ID" json:"company_id"`
	RoleID     int64     `gorm:"column:role_id;index;comment:角色ID" json:"role_id"`
	Permission string    `gorm:"column:permission;size:64;not null;comment:权限点 resource:action" json:"permission"`
	CreatedAt  time.Time `gorm:"column:created_at;comment:创建时间" json:"created_at"`
}

func (SysRolePermission) TableName() string { return "sys_role_permission" }

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
	ReceiverID   int64 `gorm:"column:receiver_id;index;comment:接收人ID" json:"receiver_id"`
	ReceiverType int   `gorm:"column:receiver_type;type:tinyint;default:1;comment:接收人类型 1-员工 2-客户" json:"receiver_type"`
	// Type 与 DDL 一致为 tinyint（1-订单 2-财务 3-系统）：
	// 历史实现用 string 承载并列的 "order"/"finance" 字面量，严格模式下每次写通知
	// 都会被 MySQL 拒收（Incorrect integer value），且失败只 Warnf —— 站内通知实际从未落库。
	Type    enum.NotificationType `gorm:"column:type;type:tinyint;comment:类型 1-订单 2-财务 3-系统" json:"type"`
	Title   string                `gorm:"column:title;size:100;comment:标题" json:"title"`
	Content string                `gorm:"column:content;size:500;comment:内容" json:"content"`
	BizType string                `gorm:"column:biz_type;size:20;comment:业务类型 order-订单 refund-退款" json:"biz_type"`
	BizID   int64                 `gorm:"column:biz_id;comment:业务ID" json:"biz_id"`
	IsRead  int                   `gorm:"column:is_read;type:tinyint;default:0;comment:是否已读 0-未读 1-已读" json:"is_read"`
	ReadAt  *string               `gorm:"column:read_at;comment:已读时间" json:"read_at"`
}

func (SysNotification) TableName() string { return "sys_notification" }

// SysFeedback 员工意见反馈（小程序「我的 → 意见反馈」提交）。
// 自助类数据：user_id 即提交人（sys_user.id），按 company_id 隔离租户。
// 处理动作（status/remark/handled_at）在管理后台维护，员工端只写不改。
type SysFeedback struct {
	TenantBase
	UserID    int64   `gorm:"column:user_id;index;comment:提交人ID(员工, sys_user.id)" json:"user_id"`
	Type      string  `gorm:"column:type;size:20;comment:问题类型" json:"type"`
	Status    int     `gorm:"column:status;type:tinyint;default:1;comment:处理状态 1-待处理 2-已处理" json:"status"`
	Content   string  `gorm:"column:content;size:1000;comment:问题描述" json:"content"`
	Images    string  `gorm:"column:images;size:1000;comment:截图URL(逗号分隔)" json:"images"`
	Contact   string  `gorm:"column:contact;size:100;comment:联系方式" json:"contact"`
	HandledAt *string `gorm:"column:handled_at;comment:处理时间" json:"handled_at"`
	Remark    string  `gorm:"column:remark;size:500;comment:处理备注" json:"remark"`
}

func (SysFeedback) TableName() string { return "sys_feedback" }
