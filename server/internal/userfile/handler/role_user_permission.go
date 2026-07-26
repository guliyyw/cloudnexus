package handler

import (
	"net/http"
	"strconv"

	"github.com/cloudnexus/server/pkg/httputil"
	"github.com/cloudnexus/server/pkg/response"
	"github.com/gin-gonic/gin"
)

type replaceUserPermissionsReq struct {
	PermissionIDs []string `json:"permission_ids"`
}

func (h *RoleHandler) HandleGetUserPermissions(c *gin.Context) {
	userID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "Invalid user ID"))
		return
	}
	perms, err := h.svc.GetUserPermissions(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(500, "Failed to get user permissions"))
		return
	}
	c.JSON(http.StatusOK, response.OKWithData(perms))
}

func (h *RoleHandler) HandleReplaceUserPermissions(c *gin.Context) {
	userID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "Invalid user ID"))
		return
	}
	var req replaceUserPermissionsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "Invalid permission list"))
		return
	}
	permissionIDs := make([]uint64, 0, len(req.PermissionIDs))
	for _, rawID := range req.PermissionIDs {
		permissionID, parseErr := strconv.ParseUint(rawID, 10, 64)
		if parseErr != nil {
			c.JSON(http.StatusBadRequest, response.Error(400, "Invalid permission ID"))
			return
		}
		permissionIDs = append(permissionIDs, permissionID)
	}
	if err := h.svc.ReplaceUserPermissions(c.GetUint64("user_id"), userID, permissionIDs); err != nil {
		httputil.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OK("Permissions updated"))
}
