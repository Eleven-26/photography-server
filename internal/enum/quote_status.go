package enum

// QuoteStatus 报价单状态
type QuoteStatus int

const (
	QuoteStatusDraft     QuoteStatus = 1 // 草稿
	QuoteStatusSent      QuoteStatus = 2 // 已发送
	QuoteStatusAccepted  QuoteStatus = 3 // 已接受
	QuoteStatusRejected  QuoteStatus = 4 // 已拒绝
	QuoteStatusConverted QuoteStatus = 5 // 已成交
)

var quoteStatusName = map[QuoteStatus]string{
	QuoteStatusDraft:     "草稿",
	QuoteStatusSent:      "已发送",
	QuoteStatusAccepted:  "已接受",
	QuoteStatusRejected:  "已拒绝",
	QuoteStatusConverted: "已成交",
}

func QuoteStatusName(status QuoteStatus) string {
	if name, ok := quoteStatusName[status]; ok {
		return name
	}
	return "未知"
}
