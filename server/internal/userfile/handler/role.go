package handler

import (
	"net/http"
	"strconv"

	"github.com/cloudnexus/server/internal/userfile/service"
	"github.com/cloudnexus/server/pkg/httputil"
	"github.com/cloudnexus/server/pkg/response"
	"github.com/gin-gonic/gin"
)

type RoleHandler struct {
	svc *service.RoleService
}

func NewRoleHandler(svc *service.RoleService) *RoleHandler {
	return &RoleHandler{svc: svc}
}

func (h *RoleHandler) HandleListRoles(c *gin.Context) {
	roles, err := h.svc.ListRoles()
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(500, "获取角色列表失败"))
		return
	}
	c.JSON(http.StatusOK, response.OKWithData(roles))
}

type createRoleReq struct {
	Name        string `json:"name" binding:"required"`
	Code        string `json:"code" binding:"required"`
	Description string `json:"description"`
}

func (h *RoleHandler) HandleCreateRole(c *gin.Context) {
	var req createRoleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "参数错误"))
		return
	}
	role, err := h.svc.CreateRole(req.Name, req.Code, req.Description)
	if err != nil {
		httputil.HandleError(c, err)
		return
	}
	c.JSON(http.StatusCreated, response.OKWithData(role))
}

type updateRoleReq struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (h *RoleHandler) HandleUpdateRole(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "无效的角色 ID"))
		return
	}
	var req updateRoleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "参数错误"))
		return
	}
	role, err := h.svc.UpdateRole(id, req.Name, req.Description)
	if err != nil {
		httputil.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OKWithData(role))
}

func (h *RoleHandler) HandleDeleteRole(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "无效的角色 ID"))
		return
	}
	if err := h.svc.DeleteRole(id); err != nil {
		httputil.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OK("已删除"))
}

func (h *RoleHandler) HandleListPermissions(c *gin.Context) {
	perms, err := h.svc.ListPermissions()
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(500, "获取权限列表失败"))
		return
	}
	c.JSON(http.StatusOK, response.OKWithData(perms))
}

type assignPermsReq struct {
	PermissionIDs []uint64 `json:"permission_ids" binding:"required"`
}

func (h *RoleHandler) HandleAssignPermissions(c *gin.Context) {
	roleID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "无效的角色 ID"))
		return
	}
	var req assignPermsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "参数错误"))
		return
	}
	if err := h.svc.AssignRolePermissions(roleID, req.PermissionIDs); err != nil {
		httputil.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OK("已分配"))
}

type assignUserRoleReq struct {
	RoleID uint64 `json:"role_id,string" binding:"required"`
}

func (h *RoleHandler) HandleGetUserRoles(c *gin.Context) {
	userID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "无效的用户 ID"))
		return
	}
	roles, err := h.svc.GetUserRoles(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(500, "获取用户角色失败"))
		return
	}
	c.JSON(http.StatusOK, response.OKWithData(roles))
}

func (h *RoleHandler) HandleAssignUserRole(c *gin.Context) {
	operatorID := c.GetUint64("user_id")
	userID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "无效的用户 ID"))
		return
	}
	var req assignUserRoleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "参数错误"))
		return
	}
	if err := h.svc.AssignUserRole(operatorID, userID, req.RoleID); err != nil {
		httputil.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OK("已分配"))
}

func (h *RoleHandler) HandleRemoveUserRole(c *gin.Context) {
	userID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "无效的用户 ID"))
		return
	}
	roleID, err := strconv.ParseUint(c.Param("roleId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "无效的角色 ID"))
		return
	}
	if err := h.svc.RemoveUserRole(userID, roleID); err != nil {
		httputil.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OK("已移除"))
}
