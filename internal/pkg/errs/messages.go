package errs

// 错误文案常量表。
//
// 约定：业务代码里的错误提示一律引用本文件的常量（errs.BadRequest(errs.ErrXxx)），
// 不得就地写字面量 —— 文案要改只改这里一处，也便于前端按文案做映射。
// 传空串 errs.BadRequest("") 表示「使用该错误类型的默认文案」，属有意写法，不要替换成常量。

// ======================== 通用 ========================

const (
	ErrNotFound     = "记录不存在"
	ErrBadRequest   = "请求参数错误"
	ErrInternal     = "系统内部错误"
	ErrUnauthorized = "未授权"
	ErrForbidden    = "无权限操作"
	ErrDuplicate    = "记录已存在"
)

const (
	ErrParamInvalid              = "参数错误"
	ErrNoFieldsToUpdate          = "未提供任何需要变更的字段"
	ErrApproveConclusionRequired = "缺少审批结论 approved"
	ErrTimeSlotInvalid           = "时间段无效"
	ErrWeekdayInvalid            = "星期参数错误（0-周日 ... 6-周六）"
	ErrDateRequired              = "请选择日期"
	ErrDateFormatInvalid         = "日期格式错误，应为 2006-01-02"
	ErrDateInPast                = "不可选择过去的日期"
)

// ======================== 文件上传 ========================

const (
	ErrUploadInvalidFile   = "仅支持图片/视频文件"
	ErrFileTypeUnsupported = "不支持的文件类型"
	ErrFileTooLarge        = "文件大小超过限制"
	ErrFileRequired        = "请上传文件"
	ErrFileReadFailed      = "文件读取失败"
)

// ======================== 认证/账号 ========================

const (
	ErrAccountDisabled          = "账号已被停用"
	ErrAccountWrong             = "账号或密码错误"
	ErrLoginExpired             = "登录已失效，请重新登录"
	ErrAccountStatusAbnormal    = "账号状态异常"
	ErrClientAccountUnavailable = "账号不可用，请联系工作室"
	ErrClientAccountNotFound    = "账号不存在，请联系工作室开通"
	ErrLoginTooManyAttempts     = "登录失败次数过多，请 15 分钟后再试"
	ErrAccountLocked            = "账号已被锁定，请 15 分钟后再试"
	ErrMobileFormatInvalid      = "手机号格式错误"
	ErrMobileFormatWrong        = "手机号格式不正确"
	ErrMobileUnchanged          = "新手机号与当前手机号相同"
	ErrMobileTaken              = "该手机号已被其他账号使用"
	ErrStaffMobileUnbound       = "当前账号未绑定手机号，请联系管理员"
	ErrSmsTooFrequent           = "发送过于频繁，请稍后再试"
	ErrSmsDailyLimit            = "今日发送次数已达上限"
	ErrSmsCodeExpired           = "验证码错误或已过期"
	ErrSmsCodeTooManyAttempts   = "验证码错误次数过多，请重新获取"
	ErrSmsServiceUnavailable    = "短信服务暂不可用，请稍后再试"
)

// ======================== 用户 ========================

const (
	ErrUserNotFound   = "用户不存在"
	ErrUserDuplicate  = "用户名已存在"
	ErrUserSelfDelete = "不能删除当前登录账号"
	ErrPasswordWrong  = "原密码错误"
)

// ======================== 角色/门店 ========================

const (
	ErrRoleNotFound  = "角色不存在"
	ErrStoreNotFound = "门店不存在"
)

// ======================== 权限（RBAC） ========================

const (
	ErrPermInvalid      = "权限点不合法"
	ErrDataScopeInvalid = "数据范围不合法（1-全部 2-本门店 3-仅本人）"
	ErrAdminRoleLocked  = "内置超管角色权限固定，不支持修改"
	ErrRoleInUse        = "该角色下仍有成员，请先调整成员角色"
)

// ======================== 客户 ========================

const (
	ErrCustomerNotFound  = "客户不存在"
	ErrCustomerDuplicate = "客户已存在"
)

const (
	ErrCustomerNotInCompany          = "客户不存在或不属于当前工作室"
	ErrCustomerNameRequired          = "姓名不能为空"
	ErrCustomerNameTooLong           = "姓名过长"
	ErrCustomerWechatTooLong         = "微信号过长"
	ErrCustomerGenderInvalid         = "性别取值不合法"
	ErrCustomerBirthdayFormatInvalid = "生日格式应为 YYYY-MM-DD"
	ErrCustomerPreferStyleTooLong    = "偏好风格过长"
	ErrCustomerPreferSceneTooLong    = "常用场景过长"
)

// ======================== 线索/报价 ========================

const (
	ErrLeadNotFound  = "线索不存在"
	ErrQuoteNotFound = "报价单不存在"
)

const (
	ErrLeadMobileRequired    = "线索缺少手机号，无法建立客户，请先补全手机号"
	ErrQuoteStatusRequired   = "报价状态不能为空"
	ErrQuoteFeedbackRequired = "修改意见不能为空"
	ErrQuoteNoLead           = "该报价未关联线索，请联系工作室"
)

// ======================== 定制需求 ========================

const (
	ErrCustomRequestNotFound        = "定制需求不存在"
	ErrCustomRequestPackageRequired = "请选择套餐后再转订单"
	ErrCustomRequestClosed          = "定制需求已关闭，不可转订单"
)

// ======================== 套餐 ========================

const (
	ErrPackageNotFound      = "套餐不存在"
	ErrPackageActiveDelete  = "已上架套餐不可删除，请先下线"
	ErrPackageActiveUpdate  = "已上架套餐不可编辑，请先下线"
	ErrPackageStatusInvalid = "套餐状态不合法（1-草稿 2-已上架 3-已下线）"
	ErrPackageOffline       = "套餐已下架"
	ErrPackageOfflineOrder  = "套餐已下架，不可下单"
)

// ======================== 订单/加项 ========================

const (
	ErrOrderNotFound      = "订单不存在"
	ErrOrderStatusInvalid = "订单状态不允许流转"
	ErrOrderCompleted     = "已完成或已取消的订单不可修改"
	ErrOrderNoCustomer    = "请选择客户或来源线索"
)

const (
	ErrOrderIDInvalid          = "订单ID无效"
	ErrOrderForbidden          = "无权操作该订单"
	ErrOrderClosedAddon        = "订单已完成或已取消，不可修改加项"
	ErrOrderPackageRequired    = "请选择套餐"
	ErrOrderShootDateRequired  = "请选择拍摄日期与时段"
	ErrOrderCancelNotAllowed   = "当前状态不可取消，如需取消请联系工作室"
	ErrOrderInShooting         = "订单已进入拍摄流程，需求变更请联系工作室"
	ErrOrderRequestChangeEmpty = "请至少填写一项需要修改的需求"
	ErrOrderNoPreparation      = "该订单暂无拍前准备内容"
	ErrOrderNoReceivable       = "订单已无待收金额，无需登记"
	ErrAddonNotFound           = "加项不存在"
	ErrAddonAmountNegative     = "加选金额不能为负"
)

// ======================== 预约下单（客户端） ========================

const (
	ErrBookingPaused              = "工作室已暂停接单，请稍后再试"
	ErrBookingSlotTaken           = "该时段已被预约，请选择其他时段"
	ErrBookingProjectTypeRequired = "请选择拍摄类型"
	ErrBookingMobileRequired      = "请填写联系电话"
	ErrBookingStoreInvalid        = "门店不存在或不属于当前机构"
	ErrBookingStaffNotAvailable   = "所选摄影师不存在或已停用"
	ErrBookingStaffNotInStore     = "所选摄影师不属于该门店"
)

// ======================== 改期 ========================

const (
	ErrRescheduleNotFound           = "改期单不存在"
	ErrRescheduleForbidden          = "无权操作该改期单"
	ErrRescheduleDateRequired       = "请选择新的拍摄日期与时段"
	ErrRescheduleOrderStatusInvalid = "当前订单状态不可改期"
	ErrReschedulePendingClient      = "已有待确认的改期申请，请耐心等待"
	ErrReschedulePendingStaff       = "已有待确认的改期申请，请先处理"
	ErrRescheduleTooLateClient      = "距拍摄不足24小时，不可改期，请联系工作室"
	ErrRescheduleTooLateStaff       = "距拍摄不足 24 小时，不可改期，请与客户协商"
	ErrRescheduleHandled            = "该改期单已处理"
	ErrRescheduleHandledWithdraw    = "该改期单已处理，不可撤回"
	ErrRescheduleFeeNotDue          = "改期申请尚未通过，暂无需支付调度费"
	ErrRescheduleFeeFree            = "本次改期无需支付调度费"
	ErrRescheduleFeeVoucherPending  = "调度费凭证已提交，请等待工作室核验"
	ErrRescheduleFeePaid            = "调度费已核验到账，无需重复支付"
)

// ======================== 交付 ========================

const (
	ErrDeliveryNotFound     = "交付单不存在"
	ErrDeliveryStageInvalid = "当前阶段不可操作"
)

const (
	ErrDeliveryFileNotFound      = "交付文件不存在"
	ErrDeliveryNoFeedback        = "该文件没有客户反馈"
	ErrDeliveryStageParamInvalid = "交付阶段参数错误"
)

// ======================== 选片/加片/成片 ========================

const (
	ErrExtraLocked             = "已确认加片费用，选片已锁定；如需调整请联系工作室"
	ErrSelectionClosed         = "选片已截止，如需调整请联系工作室"
	ErrExtraStatusInvalid      = "加片确认状态异常，请刷新后重试"
	ErrConfirmStageInvalid     = "当前阶段不可确认成片"
	ErrRetouchFeedbackRequired = "请填写修改意见"
)

// ======================== 收款/退款 ========================

const (
	ErrPaymentNotFound       = "收款记录不存在"
	ErrPaymentConfirmed      = "该收款已核验"
	ErrPaymentMethodNotFound = "收款方式不存在"
)

const (
	ErrPaymentAmountPositive          = "收款金额必须大于 0"
	ErrPaymentTypeInvalid             = "收款类型不合法（deposit/final/addon）"
	ErrPaymentExceedRemaining         = "收款金额超过订单剩余应收"
	ErrPaymentExceedRemainingRegister = "登记金额超过订单剩余应收，请核对后重试"
	ErrPaymentConfirmExceed           = "确认后收款将超过订单剩余应收，请核对金额"
	ErrPaymentConfirmedDelete         = "已确认的收款不可删除，请走退款流程"
	ErrPaymentMethodDisabled          = "该收款方式已停用，请选择其他方式"
	ErrBankVoucherRequired            = "银行转账请上传转账凭证"
)

const (
	ErrRefundNotFound      = "退款单不存在"
	ErrRefundProcessed     = "该退款单已处理"
	ErrRefundZero          = "已无可退金额"
	ErrRefundNoTime        = "按退款规则当前可退金额为0（距离拍摄不足24小时或未付定金）"
	ErrRefundCancelled     = "订单已取消，请直接走已取消流程"
	ErrRefundNotPending    = "退款尚未通过或已驳回，暂不可确认收款"
	ErrRefundExceedAmount  = "退款金额超过订单剩余可退金额"
	ErrRefundApproveExceed = "退款金额超过订单剩余可退金额，无法通过"
	ErrRefundExists        = "该订单已有申请中的退款单，请先处理"
)

// ======================== 评价 ========================

const (
	ErrReviewOrderNotComplete = "订单完成后方可评价"
	ErrReviewRatingInvalid    = "评分需在 1-5 之间"
	ErrReviewExists           = "该订单已评价过"
)

// ======================== 作品 ========================

const (
	ErrAssetNotFound          = "作品不存在"
	ErrAssetNotFoundOrPrivate = "作品不存在或未公开"
	ErrAssetStatusInvalid     = "作品状态取值非法"
	ErrAssetVisibilityInvalid = "可见性取值非法"
	ErrAssetFeaturedInvalid   = "精选取值非法"
)

// ======================== 反馈 ========================

const (
	ErrFeedbackContentRequired = "请填写问题描述"
	ErrFeedbackContentTooLong  = "问题描述过长（最多 1000 字）"
	ErrFeedbackTypeInvalid     = "问题类型不合法"
)

// ======================== 员工端 ========================

const (
	ErrMessageContentRequired = "消息内容不能为空"
	ErrReplyContentRequired   = "回复内容不能为空"
	ErrBriefNotFound          = "简报项不存在"
	ErrBriefHandled           = "该简报项已处理"
	ErrBriefConfirmRequired   = "请填写确认内容"
	ErrSlotTemplateNotFound   = "模板不存在"
	ErrSlugTooLong            = "主页标识不能超过 50 个字符"
	ErrSlugFormatInvalid      = "主页标识只能包含小写字母、数字和连字符，且不能以连字符开头"
)

// ======================== 分享链接 ========================

const (
	ErrSlugRequired          = "缺少工作室标识（slug）"
	ErrHomepageNotConfigured = "预约主页不存在或未配置短链标识，请联系工作室"
)

// ======================== 基础设施/调试 ========================

const (
	ErrRedisNotConnected   = "redis 未连接"
	ErrMongoNotConnected   = "mongodb 未连接"
	ErrNatsNotConnected    = "nats 未连接"
	ErrESNotConnected      = "elasticsearch 未连接"
	ErrJetStreamDisabled   = "jetStream 未启用"
	ErrJaegerDisabled      = "jaeger 未启用（enable=false 或初始化失败）"
	ErrJaegerNotEnabled    = "jaeger 未启用"
	ErrConfigSecretMissing = "未配置主密钥（APP_CONFIG_SECRET / APP_CONFIG_SECRET_FILE），无法加密"
)

// ======================== 其他 ========================

const (
	ErrCompanyNotFound  = "公司信息不存在"
	ErrCalendarNotFound = "档期不存在"
)
