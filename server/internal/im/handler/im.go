package handler

import (
	"net/http"
	"strconv"

	"github.com/cloudnexus/server/internal/im/service"
	apperrors "github.com/cloudnexus/server/pkg/errors"
	"github.com/cloudnexus/server/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type IMHandler struct {
	svc *service.IMService
	hub *service.Hub
}

func NewIMHandler(svc *service.IMService, hub *service.Hub) *IMHandler {
	return &IMHandler{svc: svc, hub: hub}
}

func (h *IMHandler) HandleWebSocket(c *gin.Context) {
	userID := c.GetUint64("user_id")
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	h.hub.Register(userID, conn)
}

func (h *IMHandler) HandleGetConversations(c *gin.Context) {
	userID := c.GetUint64("user_id")
	convs, err := h.svc.GetConversations(userID)
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OKWithData(convs))
}

type createConvReq struct {
	Type      string   `json:"type" binding:"required"`
	Name      string   `json:"name"`
	MemberIDs []string `json:"member_ids" binding:"required"`
}

func (h *IMHandler) HandleCreateConversation(c *gin.Context) {
	userID := c.GetUint64("user_id")
	var req createConvReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "参数错误: "+err.Error()))
		return
	}

	if req.Type == "private" && len(req.MemberIDs) > 0 {
		targetID, err := strconv.ParseUint(req.MemberIDs[0], 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, response.Error(400, "无效的用户 ID"))
			return
		}
		conv, err := h.svc.CreatePrivateConversation(userID, targetID)
		if err != nil {
			handleError(c, err)
			return
		}
		c.JSON(http.StatusCreated, response.OKWithData(conv))
		return
	}

	if req.Type == "group" {
		memberIDs := make([]uint64, 0, len(req.MemberIDs))
		for _, s := range req.MemberIDs {
			id, err := strconv.ParseUint(s, 10, 64)
			if err != nil {
				c.JSON(http.StatusBadRequest, response.Error(400, "无效的用户 ID: "+s))
				return
			}
			memberIDs = append(memberIDs, id)
		}
		conv, err := h.svc.CreateGroupConversation(userID, req.Name, memberIDs)
		if err != nil {
			handleError(c, err)
			return
		}
		c.JSON(http.StatusCreated, response.OKWithData(conv))
		return
	}

	c.JSON(http.StatusBadRequest, response.Error(400, "不支持的会话类型"))
}

// --- Group handlers ---

func (h *IMHandler) HandleGetGroupMembers(c *gin.Context) {
	userID := c.GetUint64("user_id")
	convID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "无效的会话ID"))
		return
	}
	members, err := h.svc.GetGroupMembers(convID, userID)
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OKWithData(members))
}

type addMemberReq struct {
	UserID string `json:"user_id" binding:"required"`
}

func (h *IMHandler) HandleAddGroupMember(c *gin.Context) {
	userID := c.GetUint64("user_id")
	convID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "无效的会话ID"))
		return
	}
	var req addMemberReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "参数错误"))
		return
	}
	targetID, err := strconv.ParseUint(req.UserID, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "无效的用户ID"))
		return
	}
	if err := h.svc.AddGroupMember(userID, convID, targetID); err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OK("已添加"))
}

func (h *IMHandler) HandleRemoveGroupMember(c *gin.Context) {
	userID := c.GetUint64("user_id")
	convID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "无效的会话ID"))
		return
	}
	targetID, err := strconv.ParseUint(c.Param("uid"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "无效的用户ID"))
		return
	}
	if err := h.svc.RemoveGroupMember(userID, convID, targetID); err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OK("已移除"))
}

func (h *IMHandler) HandleLeaveGroup(c *gin.Context) {
	userID := c.GetUint64("user_id")
	convID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "无效的会话ID"))
		return
	}
	if err := h.svc.LeaveGroup(userID, convID); err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OK("已退出群聊"))
}

// --- Friend handlers ---

type sendFriendReq struct {
	FriendName string `json:"friend_name" binding:"required"`
}

func (h *IMHandler) HandleSendFriendRequest(c *gin.Context) {
	userID := c.GetUint64("user_id")
	var req sendFriendReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "参数错误"))
		return
	}
	friend, err := h.svc.SendFriendRequest(userID, req.FriendName)
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusCreated, response.OKWithData(friend))
}

func (h *IMHandler) HandleAcceptRequest(c *gin.Context) {
	userID := c.GetUint64("user_id")
	requestID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "无效的请求ID"))
		return
	}
	conv, err := h.svc.AcceptFriendRequest(requestID, userID)
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OKWithData(conv))
}

func (h *IMHandler) HandleRejectRequest(c *gin.Context) {
	userID := c.GetUint64("user_id")
	requestID, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if err := h.svc.RejectFriendRequest(requestID, userID); err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OK("rejected"))
}

func (h *IMHandler) HandleListFriends(c *gin.Context) {
	userID := c.GetUint64("user_id")
	friends, err := h.svc.ListFriends(userID)
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OKWithData(friends))
}

func (h *IMHandler) HandleListPendingRequests(c *gin.Context) {
	userID := c.GetUint64("user_id")
	requests, err := h.svc.ListPendingRequests(userID)
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OKWithData(requests))
}

func (h *IMHandler) HandleRemoveFriend(c *gin.Context) {
	userID := c.GetUint64("user_id")
	friendID, _ := strconv.ParseUint(c.Param("friend_id"), 10, 64)
	if err := h.svc.RemoveFriend(userID, friendID); err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OK("removed"))
}

func (h *IMHandler) HandleDeleteConversation(c *gin.Context) {
	userID := c.GetUint64("user_id")
	convID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "无效的会话ID"))
		return
	}
	if err := h.svc.DeleteConversation(userID, convID); err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OK("deleted"))
}

func (h *IMHandler) HandleGetMessages(c *gin.Context) {
	userID := c.GetUint64("user_id")
	convID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "无效的会话ID"))
		return
	}
	before, _ := strconv.ParseUint(c.DefaultQuery("before", "0"), 10, 64)
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))

	msgs, err := h.svc.GetMessages(convID, userID, before, limit)
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OKWithData(msgs))
}

func handleError(c *gin.Context, err error) {
	if appErr, ok := err.(*apperrors.AppError); ok {
		c.JSON(appErr.Code, response.Error(appErr.Code, appErr.Message))
		return
	}
	c.JSON(http.StatusInternalServerError, response.Error(500, "服务器内部错误"))
}
