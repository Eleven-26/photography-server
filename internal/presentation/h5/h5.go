// Package h5 客户 H5 端接口（客户预约全链路：浏览套餐 → 提交预约 → 支付定金 →
// 选片 → 确认成片 → 评价）。公开接口无需登录，业务接口经 CustomerAuth 注入客户上下文。
//
// 路由注册自 2026-09-16 起**不在本包**：客户区 48 条路由声明于
// presentation/routes/client.go 的 ClientPublic / ClientAuthed，由 router 按端与鉴权分组挂载，
// 且 H5 与小程序客户区引用**同一份 []Route**（同一 Handler 函数指针）。
// 本文件只承载 handler 实现与端内辅助（slugFrom / staffFrom / requireCompany）。
package h5

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"photography-server/internal/config"
	"photography-server/internal/contract"
	"photography-server/internal/middleware"
	"photography-server/internal/pkg/errs"
	"photography-server/internal/pkg/params"
	"photography-server/internal/pkg/response"
	"photography-server/internal/presentation/bind"
	"photography-server/internal/service"
)

// Controller 客户 H5 端接口（客户预约全链路：浏览套餐 → 提交预约 → 支付定金 →
// 选片 → 确认成片 → 评价）。公开接口无需登录，业务接口经 CustomerAuth 注入客户上下文。
type Controller struct {
	Svc *service.Service
	Cfg *config.Config
}

func New(svc *service.Service, cfg *config.Config) *Controller {
	return &Controller{Svc: svc, Cfg: cfg}
}

// slugFrom 提取客户端公开接口的预约主页短链标识：query slug 与 X-Slug 头二选一（头优先）。
// slug 形如 "sunset-studio"，为工作室预约主页 URL/二维码携带的不可枚举标识（#29），
// 服务端据此反查 company_id，绝不接受客户端直传裸 company_id。
func slugFrom(c *gin.Context) string {
	if v := c.GetHeader("X-Slug"); v != "" {
		return v
	}
	// POST 参数统一走 body；query 仅作预约主页短链的兜底
	if v := params.Str(c, "slug"); v != "" {
		return v
	}
	return c.Query("slug")
}

// staffFrom 提取分享链接携带的员工账号 ID：X-Staff-Id 头 → body staff_id → query ?staff_id=
// （与 slugFrom 同款优先级）。链接形如 https://host/?slug=xxx&staff_id=12，由员工端
// 「我的预约主页」分享出去（见 contract.NewStaffStudioSettingResp）。
// 用途：客户从谁的链接进来下单，订单就归到该员工名下（biz_order.photographer_id），
// 员工端「仅本人」数据范围据此能查到自己的客户单。
// 缺失或非法一律返回 0 —— 视为「非分享进入」，不报错，订单由工作室后续指派。
func staffFrom(c *gin.Context) int64 {
	candidates := []string{c.GetHeader("X-Staff-Id"), params.Str(c, "staff_id"), c.Query("staff_id")}
	for _, raw := range candidates {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		if id, err := strconv.ParseInt(raw, 10, 64); err == nil && id > 0 {
			return id
		}
	}
	return 0
}

// requireCompany 按 slug 反查并校验租户（数据库不存在/未配置 slug 时返回业务错误）
func (h *Controller) requireCompany(c *gin.Context) (int64, error) {
	slug := slugFrom(c)
	if slug == "" {
		return 0, errs.BadRequest(errs.ErrSlugRequired)
	}
	companyID, err := h.Svc.ResolveCompanyBySlug(c.Request.Context(), slug)
	if err != nil {
		return 0, errs.Internal("")
	}
	if companyID <= 0 {
		return 0, errs.BadRequest(errs.ErrHomepageNotConfigured)
	}
	return companyID, nil
}

// CustomRequestSubmit 提交定制需求（游客/登录均可）。
// 摄影师归属（2026-09-15 补齐）：body.photographer_id（客户在 H5 定制需求页的显式选择）优先，
// 分享链接的 staff_id 兜底 —— 两者都缺则落门店/公共池，见 service.ClientSubmitCustomRequest。
// @Summary      提交定制需求
// @Description  提交定制需求，**游客与登录客户均可**。已登录取令牌内公司，游客按 slug 反查租户。
// @Description  摄影师归属：body.photographer_id（客户显式选择）优先，分享链接的 staff_id 兜底，两者都缺则落门店/公共池。
// @Description  ⚠️ 本接口会往 crm_customer 建档（手机号唯一键），公开可调，存在被刷数据的风险。
// @Tags         客户区·定制需求
// @Accept       json
// @Produce      json
// @Param        req  body  contract.ClientCustomRequestReq  true  "定制需求信息"
// @Success      200  {object}  response.Body{data=model.CustomRequest}
// @Failure      400  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Router       /h5/custom-request/submit [post]
func (h *Controller) CustomRequestSubmit(c *gin.Context) {
	var req contract.ClientCustomRequestReq
	if err := bind.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	cu := middleware.GetClientUser(c)
	var companyID int64
	if cu != nil {
		// 已登录：租户取令牌内绑定的公司（可信，不信任客户端请求参数）
		companyID = cu.CompanyID
	} else {
		// 游客：按 slug 反查（#29）
		var err error
		companyID, err = h.requireCompany(c)
		if err != nil {
			response.Fail(c, err)
			return
		}
	}
	m, err := h.Svc.ClientSubmitCustomRequest(c.Request.Context(), companyID, cu, req, staffFrom(c))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, m)
}

// CustomRequestList 我的定制需求
// @Summary      我的定制需求
// @Description  客户本人提交过的定制需求历史（分页）。
// @Tags         客户区·定制需求
// @Accept       json
// @Produce      json
// @Param        req  body  contract.PageReq  true  "分页参数（page / page_size）"
// @Success      200  {object}  response.Body{data=response.Page{list=[]model.CustomRequest,total=int,page=int,page_size=int}}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /h5/custom-request/list [post]
func (h *Controller) CustomRequestList(c *gin.Context) {
	cu := middleware.GetClientUser(c)
	page, pageSize := bind.Pager(c)
	list, total, err := h.Svc.ClientCustomRequests(c.Request.Context(), cu, page, pageSize)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.PageOK(c, list, total, page, pageSize)
}
