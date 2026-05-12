package handler

import (
	"net/http"
	"strconv"

	"github.com/cloudnexus/server/internal/im/service"
	"github.com/cloudnexus/server/pkg/response"
	"github.com/gin-gonic/gin"
)

type FriendEnhanceHandler struct {
	svc        *service.IMService
	blSvc      *service.BlocklistService
	presenceSvc *service.PresenceService
}

func NewFriendEnhanceHandler(svc *service.IMService, blSvc *service.BlocklistService, presenceSvc *service.PresenceService) *FriendEnhanceHandler {
	return &FriendEnhanceHandler{svc: svc, blSvc: blSvc, presenceSvc: presenceSvc}
}

// Blocklist handlers
func (h *FriendEnhanceHandler) HandleBlockUser(c *gin.Context) {
	userID := c.GetUint64("user_id")
	targetID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "无效的用户 ID"))
		return
	}
	var req struct {
		Reason string `json:"reason"`
	}
	c.ShouldBindJSON(&req)

	if err := h.blSvc.BlockUser(userID, targetID, req.Reason); err != nil {
		handleError(c, err)
		return
	}
	h.svc.RemoveFriend(userID, targetID) // also remove friend if exists
	c.JSON(http.StatusOK, response.OK("已拉黑"))
}

func (h *FriendEnhanceHandler) HandleUnblockUser(c *gin.Context) {
	userID := c.GetUint64("user_id")
	targetID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "无效的用户 ID"))
		return
	}
	if err := h.blSvc.UnblockUser(userID, targetID); err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OK("已取消拉黑"))
}

func (h *FriendEnhanceHandler) HandleGetBlocklist(c *gin.Context) {
	userID := c.GetUint64("user_id")
	list, err := h.blSvc.GetBlocklist(userID)
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OKWithData(list))
}

// Remark handler
func (h *FriendEnhanceHandler) HandleSetRemark(c *gin.Context) {
	userID := c.GetUint64("user_id")
	friendID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "无效的好友 ID"))
		return
	}
	var req struct {
		Remark string `json:"remark" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "参数错误"))
		return
	}
	if err := h.svc.SetFriendRemark(userID, friendID, req.Remark); err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OK("备注已设置"))
}

// Online status handler
func (h *FriendEnhanceHandler) HandleGetOnlineStatus(c *gin.Context) {
	idsStr := c.QueryArray("ids")
	ids := make([]uint64, 0, len(idsStr))
	for _, s := range idsStr {
		id, err := strconv.ParseUint(s, 10, 64)
		if err == nil {
			ids = append(ids, id)
		}
	}
	status := h.presenceSvc.GetOnlineStatus(ids)
	c.JSON(http.StatusOK, response.OKWithData(status))
}
