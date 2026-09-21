package wechat

// 小程序员工区接口（订单处理 + 日程 + 线索 AI 简报 + 个人中心）。
// 员工（摄影师/助理）通过小程序处理业务，身份经 StaffAuth 注入 Operator。
//
// 路由注册（2026-09-12 路由整理后）已迁到 router/endpoints.go 的 staffEndpoint：
// 与 PC 同路径的路由直接复用管理端 handler（同一函数指针，表在 routes 包），
// 本文件只保留【员工端独有 handler】与【真实端差异实现】（退款审核 approve / 创建交付单不收 body）。
// Controller 类型与 New 构造见 wechat.go。

// 路由注册已迁出本文件（2026-09-16）：
// 员工端三条登录出口（auth/sms-code、auth/login、auth/login-by-code）现声明在
// router/endpoints.go 的 staffPublicExtra —— 它们必须在拿到令牌之前可用，故单独归为
// 「免端级中间件」的路由，而不是内联在本文件里 g.POST。
// 与 PC 同路径的路由见 staffInclude，端独有的见 staffExtra；本文件从此只保留 handler。

// ---------------------------------------------------------------------
// 公开接口
// ---------------------------------------------------------------------
