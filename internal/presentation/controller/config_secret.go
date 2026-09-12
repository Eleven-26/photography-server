package controller

// 配置密文生成入口（调试路由，仅非 release 环境注册）
//
// 用途：新增/修改敏感配置字段时，直接生成 ENCv1: 密文粘贴到 Nacos 模板。
// 安全约定：
//  1. 只提供加密、**不提供解密**——避免线上出现"读取配置明文"的后门
//  2. 明文不写日志、不进入响应（响应只回密文）
//  3. 生产环境新增字段请用 CLI：go run ./cmd/configctl encrypt -v '明文'（有密钥即可，无需服务在线）

import (
	"photography-server/internal/config"
	"photography-server/internal/pkg/errs"
	"photography-server/internal/presentation/bind"
	"photography-server/internal/presentation/response"

	"github.com/gin-gonic/gin"
)

type configEncryptReq struct {
	Values []string `json:"values" binding:"required,min=1"` // 支持批量：一次加密多个字段
}

// ConfigEncrypt 生成配置密文
// POST /test/config/encrypt
func (h *Controller) ConfigEncrypt(c *gin.Context) {
	var req configEncryptReq
	if err := bind.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	cipher, err := config.LoadCipher()
	if err != nil {
		response.Fail(c, errs.Internal("加载主密钥失败: "+err.Error()))
		return
	}
	if !cipher.Enabled() {
		response.Fail(c, errs.Internal("未配置主密钥（APP_CONFIG_SECRET / APP_CONFIG_SECRET_FILE），无法加密"))
		return
	}

	items := make([]gin.H, 0, len(req.Values))
	for _, v := range req.Values {
		out, err := cipher.Encrypt(v)
		if err != nil {
			response.Fail(c, errs.Internal("加密失败: "+err.Error()))
			return
		}
		items = append(items, gin.H{"cipher": out})
	}
	response.OK(c, gin.H{"count": len(items), "items": items})
}
