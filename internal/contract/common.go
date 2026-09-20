package contract

// PageReq 通用分页请求体。
//
// 说明：本仓接口参数统一从 JSON body 读取（见 internal/pkg/params + bind.Pager），
// 该类型仅供 Swagger 文档描述只带分页参数的列表接口，字段与 bind.Pager 保持一致。
type PageReq struct {
	Page     int `json:"page"`      // 页码，默认 1
	PageSize int `json:"page_size"` // 每页条数，默认 20，上限 200
}
