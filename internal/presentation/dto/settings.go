package dto

// ======================== 请求 ========================

// CompanyUpdateReq 更新公司信息
type CompanyUpdateReq struct {
	Name         string `json:"name"`
	Logo         string `json:"logo"`
	ContactName  string `json:"contact_name"`
	ContactPhone string `json:"contact_phone"`
	Address      string `json:"address"`
}

// PaymentMethodReq 创建/更新收款方式
type PaymentMethodReq struct {
	Name        string `json:"name" binding:"required"`
	Type        string `json:"type" binding:"required"`
	AccountName string `json:"account_name"`
	AccountNo   string `json:"account_no"`
	Qrcode      string `json:"qrcode"`
	Status      int    `json:"status"`
	Sort        int    `json:"sort"`
}

// ======================== 响应 ========================

// Workspace 工作台数据
type WorkspaceResp struct {
	Company  interface{} `json:"company"`
	Stores   interface{} `json:"stores"`
	Roles    interface{} `json:"roles"`
	Payments interface{} `json:"payment_methods"`
}
