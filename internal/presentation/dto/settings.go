package dto

// ======================== 请求 ========================

// CompanyUpdateReq 更新公司信息
type CompanyUpdateReq struct {
	Name         string `json:"name"`          // 公司名称
	Logo         string `json:"logo"`          // 公司Logo URL
	ContactName  string `json:"contact_name"`  // 联系人姓名
	ContactPhone string `json:"contact_phone"` // 联系人电话
	Address      string `json:"address"`       // 公司地址
}

// PaymentMethodReq 创建/更新收款方式
type PaymentMethodReq struct {
	Name        string `json:"name" binding:"required"` // 收款方式名称
	Type        string `json:"type" binding:"required"` // 类型: alipay/wechat/bank/cash
	AccountName string `json:"account_name"`            // 账户名称
	AccountNo   string `json:"account_no"`              // 账号
	Qrcode      string `json:"qrcode"`                  // 收款码URL
	Status      int    `json:"status"`                  // 状态: 1启用 0禁用
	Sort        int    `json:"sort"`                    // 排序
}

// ======================== 响应 ========================

// WorkspaceResp 工作台数据
type WorkspaceResp struct {
	Company  interface{} `json:"company"`         // 公司信息
	Stores   interface{} `json:"stores"`          // 门店列表
	Roles    interface{} `json:"roles"`           // 角色列表
	Payments interface{} `json:"payment_methods"` // 收款方式列表
}
