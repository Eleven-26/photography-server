package controller

import (
	"github.com/gin-gonic/gin"

	"photography-server/internal/contract"
	"photography-server/internal/middleware"
	"photography-server/internal/pkg/params"
	"photography-server/internal/pkg/response"
	"photography-server/internal/presentation/bind"
)

// @Summary      员工列表
// @Description  分页查询员工账号，支持关键词与门店过滤。
// @Tags         用户与角色
// @Accept       json
// @Produce      json
// @Param        req  body  object{page=int,page_size=int,keyword=string,store_id=int}  true  "查询条件（store_id 0=不过滤）"
// @Success      200  {object}  response.Body{data=response.Page{list=[]model.SysUser,total=int,page=int,page_size=int}}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /user/list [post]
func (h *Controller) UserList(c *gin.Context) {
	op := middleware.GetOperator(c)
	page, pageSize := bind.Pager(c)
	storeID := params.Int64(c, "store_id")
	list, total, err := h.Svc.ListUsers(c.Request.Context(), op, page, pageSize, bind.ParamStr(c, "keyword"), storeID)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.PageOK(c, list, total, page, pageSize)
}

// @Summary      新建员工
// @Description  新建员工账号并分配角色 / 门店。
// @Tags         用户与角色
// @Accept       json
// @Produce      json
// @Param        req  body  contract.UserCreateReq  true  "员工信息"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /user/create [post]
func (h *Controller) UserCreate(c *gin.Context) {
	op := middleware.GetOperator(c)
	var req contract.UserCreateReq
	if err := bind.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.CreateUser(c.Request.Context(), op, req); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

// @Summary      更新员工
// @Description  更新员工资料与角色 / 门店；body.id 为主键。
// @Tags         用户与角色
// @Accept       json
// @Produce      json
// @Param        req  body  contract.UserUpdateReq  true  "员工信息（含 id）"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /user/update [post]
func (h *Controller) UserUpdate(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := bind.BodyID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req contract.UserUpdateReq
	if err := bind.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.UpdateUser(c.Request.Context(), op, id, req); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

// @Summary      删除员工
// @Description  删除员工账号。
// @Tags         用户与角色
// @Accept       json
// @Produce      json
// @Param        id  path  int  true  "员工ID"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /user/delete/{id} [post]
func (h *Controller) UserDelete(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.DeleteUser(c.Request.Context(), op, id); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

// @Summary      重置员工密码
// @Description  管理员为指定员工重置登录密码。
// @Tags         用户与角色
// @Accept       json
// @Produce      json
// @Param        id   path  int                       true  "员工ID"
// @Param        req  body  contract.ResetPasswordReq true  "新密码"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /user/reset-password/{id} [post]
func (h *Controller) UserResetPassword(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req contract.ResetPasswordReq
	if err := bind.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.ResetPassword(c.Request.Context(), op, id, req.Password); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

// @Summary      角色列表
// @Description  返回全部角色（不分页），含数据范围 data_scope。
// @Tags         用户与角色
// @Accept       json
// @Produce      json
// @Success      200  {object}  response.Body{data=[]model.SysRole}
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /role/list [post]
func (h *Controller) RoleList(c *gin.Context) {
	op := middleware.GetOperator(c)
	list, err := h.Svc.ListRoles(c.Request.Context(), op)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, list)
}

// @Summary      新建角色
// @Description  新建角色（权限点用 /role/grant/:id 单独保存）。
// @Tags         用户与角色
// @Accept       json
// @Produce      json
// @Param        req  body  contract.RoleCreateReq  true  "角色信息"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /role/create [post]
func (h *Controller) RoleCreate(c *gin.Context) {
	op := middleware.GetOperator(c)
	var req contract.RoleCreateReq
	if err := bind.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.CreateRole(c.Request.Context(), op, req); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

// @Summary      更新角色
// @Description  更新角色基本信息与数据范围；body.id 为主键。
// @Tags         用户与角色
// @Accept       json
// @Produce      json
// @Param        req  body  contract.RoleUpdateReq  true  "角色信息（含 id）"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /role/update [post]
func (h *Controller) RoleUpdate(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := bind.BodyID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req contract.RoleUpdateReq
	if err := bind.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.UpdateRole(c.Request.Context(), op, id, req); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

// @Summary      删除角色
// @Description  删除角色及其权限绑定。
// @Tags         用户与角色
// @Accept       json
// @Produce      json
// @Param        id  path  int  true  "角色ID"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /role/delete/{id} [post]
func (h *Controller) RoleDelete(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.DeleteRole(c.Request.Context(), op, id); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

// RoleGrant 保存角色权限（数据范围 + 权限点集合，全量覆盖式）
// @Summary      保存角色权限
// @Description  保存角色的数据范围 + 权限点集合，**全量覆盖式**（未传的权限点会被移除）。
// @Tags         用户与角色
// @Accept       json
// @Produce      json
// @Param        id   path  int                     true  "角色ID"
// @Param        req  body  contract.RoleGrantReq   true  "权限点集合与数据范围"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /role/grant/{id} [post]
func (h *Controller) RoleGrant(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req contract.RoleGrantReq
	if err := bind.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.GrantRolePerms(c.Request.Context(), op, id, req); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

// RolePerms 读取角色权限配置（权限配置界面回显）
// @Summary      读取角色权限
// @Description  读取角色的权限配置（权限配置界面回显）。
// @Tags         用户与角色
// @Accept       json
// @Produce      json
// @Param        id  path  int  true  "角色ID"
// @Success      200  {object}  response.Body{data=contract.RolePermsResp}
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /role/permissions/{id} [post]
func (h *Controller) RolePerms(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	resp, err := h.Svc.GetRolePerms(c.Request.Context(), op, id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, resp)
}

// RoleCatalog 全量权限点清单（前端渲染权限配置树的选项）
// @Summary      权限点清单
// @Description  返回全量权限点分组清单（前端渲染权限配置树的选项）。
// @Tags         用户与角色
// @Accept       json
// @Produce      json
// @Success      200  {object}  response.Body{data=[]domain.PermGroup}
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /role/catalog [post]
func (h *Controller) RoleCatalog(c *gin.Context) {
	response.OK(c, h.Svc.PermCatalog())
}

// @Summary      门店列表
// @Description  返回全部门店（不分页）。
// @Tags         用户与角色
// @Accept       json
// @Produce      json
// @Success      200  {object}  response.Body{data=[]model.SysStore}
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /store/list [post]
func (h *Controller) StoreList(c *gin.Context) {
	op := middleware.GetOperator(c)
	list, err := h.Svc.ListStores(c.Request.Context(), op)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, list)
}

// @Summary      新建门店
// @Tags         用户与角色
// @Accept       json
// @Produce      json
// @Param        req  body  contract.StoreCreateReq  true  "门店信息"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /store/create [post]
func (h *Controller) StoreCreate(c *gin.Context) {
	op := middleware.GetOperator(c)
	var req contract.StoreCreateReq
	if err := bind.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.CreateStore(c.Request.Context(), op, req); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

// @Summary      更新门店
// @Description  更新门店信息；body.id 为主键。
// @Tags         用户与角色
// @Accept       json
// @Produce      json
// @Param        req  body  contract.StoreUpdateReq  true  "门店信息（含 id）"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /store/update [post]
func (h *Controller) StoreUpdate(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := bind.BodyID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req contract.StoreUpdateReq
	if err := bind.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.UpdateStore(c.Request.Context(), op, id, req); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}

// @Summary      删除门店
// @Tags         用户与角色
// @Accept       json
// @Produce      json
// @Param        id  path  int  true  "门店ID"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Failure      500  {object}  response.Body
// @Security     BearerAuth
// @Router       /store/delete/{id} [post]
func (h *Controller) StoreDelete(c *gin.Context) {
	op := middleware.GetOperator(c)
	id, err := bind.PathID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.Svc.DeleteStore(c.Request.Context(), op, id); err != nil {
		response.Fail(c, err)
		return
	}
	response.OKNil(c)
}
