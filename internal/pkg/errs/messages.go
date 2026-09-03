package errs

// ======================== 通用 ========================

const (
	ErrNotFound     = "记录不存在"
	ErrBadRequest   = "请求参数错误"
	ErrInternal     = "系统内部错误"
	ErrUnauthorized = "未授权"
	ErrForbidden    = "无权限操作"
	ErrDuplicate    = "记录已存在"
)

// ======================== 客户 ========================

const (
	ErrCustomerNotFound  = "客户不存在"
	ErrCustomerDuplicate = "客户已存在"
)

// ======================== 线索 ========================

const (
	ErrLeadNotFound = "线索不存在"
)

// ======================== 套餐 ========================

const (
	ErrPackageNotFound     = "套餐不存在"
	ErrPackageActiveDelete = "已上架套餐不可删除，请先下线"
)

// ======================== 订单 ========================

const (
	ErrOrderNotFound      = "订单不存在"
	ErrOrderStatusInvalid = "订单状态不允许流转"
	ErrOrderCompleted     = "已完成或已取消的订单不可修改"
	ErrOrderNoCustomer    = "请选择客户或来源线索"
)

// ======================== 报价 ========================

const (
	ErrQuoteNotFound = "报价单不存在"
)

// ======================== 收款 ========================

const (
	ErrPaymentNotFound  = "收款记录不存在"
	ErrPaymentConfirmed = "该收款已核验"
)

// ======================== 退款 ========================

const (
	ErrRefundNotFound  = "退款单不存在"
	ErrRefundProcessed = "该退款单已处理"
	ErrRefundZero      = "已无可退金额"
	ErrRefundNoTime    = "按退款规则当前可退金额为0（距离拍摄不足24小时或未付定金）"
	ErrRefundCancelled = "订单已取消，请直接走已取消流程"
)

// ======================== 交付 ========================

const (
	ErrDeliveryNotFound     = "交付单不存在"
	ErrDeliveryStageInvalid = "当前阶段不可操作"
)

// ======================== 用户 ========================

const (
	ErrUserNotFound    = "用户不存在"
	ErrUserDuplicate   = "用户名已存在"
	ErrUserSelfDelete  = "不能删除当前登录账号"
	ErrPasswordWrong   = "原密码错误"
	ErrAccountDisabled = "账号已被停用"
	ErrAccountWrong    = "账号或密码错误"
)

// ======================== 角色/门店 ========================

const (
	ErrRoleNotFound  = "角色不存在"
	ErrStoreNotFound = "门店不存在"
)

// ======================== 其他 ========================

const (
	ErrCompanyNotFound       = "公司信息不存在"
	ErrPaymentMethodNotFound = "收款方式不存在"
	ErrAssetNotFound         = "作品不存在"
	ErrCalendarNotFound      = "档期不存在"
	ErrUploadInvalidFile     = "仅支持图片/视频文件"
)
