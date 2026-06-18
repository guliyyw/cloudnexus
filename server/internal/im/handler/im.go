package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/cloudnexus/server/internal/im/service"
	apperrors "github.com/cloudnexus/server/pkg/errors"
	"github.com/cloudnexus/server/pkg/middleware"
	"github.com/cloudnexus/server/pkg/model"
	"github.com/cloudnexus/server/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// isPrivateIP 检查主机名是否为内网地址
func isPrivateIP(host string) bool {
	// 检查 localhost 和常见内网主机名
	if host == "localhost" || host == "127.0.0.1" || host == "::1" {
		return true
	}
	// 解析 IP 地址
	ip := net.ParseIP(host)
	if ip == nil {
		// 可能是域名，尝试解析
		ips, err := net.LookupIP(host)
		if err != nil || len(ips) == 0 {
			return false
		}
		ip = ips[0]
	}
	// 检查是否为内网地址
	privateRanges := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"127.0.0.0/8",
		"169.254.0.0/16",
		"::1/128",
		"fc00::/7",
	}
	for _, cidr := range privateRanges {
		_, ipNet, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		if ipNet.Contains(ip) {
			return true
		}
	}
	return false
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return middleware.CheckWebSocketOrigin(r)
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

type conversationInfo struct {
	model.Conversation
	Unread      int64  `json:"unread"`
	LastMessage string `json:"last_message"`
	LastMsgType string `json:"last_msg_type"`
}

func (h *IMHandler) HandleGetConversations(c *gin.Context) {
	userID := c.GetUint64("user_id")
	convs, err := h.svc.GetConversations(userID)
	if err != nil {
		handleError(c, err)
		return
	}
	unreadMap := h.svc.GetUnreadCounts(userID)
	lastMsgMap := h.svc.GetLastMessages(userID)
	result := make([]conversationInfo, len(convs))
	for i, conv := range convs {
		info := conversationInfo{
			Conversation: conv,
			Unread:       unreadMap[conv.ID],
		}
		if lm, ok := lastMsgMap[conv.ID]; ok {
			info.LastMessage = lm.Content
			info.LastMsgType = lm.MsgType
		}
		result[i] = info
	}
	c.JSON(http.StatusOK, response.OKWithData(result))
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
	Message    string `json:"message"`
	ExpiresIn  int    `json:"expires_in"` // 0=permanent, >0=days
}

func (h *IMHandler) HandleSendFriendRequest(c *gin.Context) {
	userID := c.GetUint64("user_id")
	var req sendFriendReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "参数错误"))
		return
	}
	friend, err := h.svc.SendFriendRequest(userID, req.FriendName, req.Message, req.ExpiresIn)
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

func (h *IMHandler) HandleSearchMessages(c *gin.Context) {
	userID := c.GetUint64("user_id")
	keyword := c.Query("q")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	conversationID, _ := strconv.ParseUint(c.DefaultQuery("conversation_id", "0"), 10, 64)

	items, total, err := h.svc.SearchMessages(userID, keyword, conversationID, page, pageSize)
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OKWithData(gin.H{
		"items":           items,
		"total":           total,
		"page":            page,
		"page_size":       pageSize,
		"conversation_id": conversationID,
		"q":               keyword,
	}))
}

func (h *IMHandler) HandleGetMessageContext(c *gin.Context) {
	userID := c.GetUint64("user_id")
	convID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "无效的会话ID"))
		return
	}
	messageID, err := strconv.ParseUint(c.Param("messageId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "无效的消息ID"))
		return
	}
	before, _ := strconv.Atoi(c.DefaultQuery("before", "20"))
	after, _ := strconv.Atoi(c.DefaultQuery("after", "20"))

	msgs, err := h.svc.GetMessageContext(convID, messageID, userID, before, after)
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OKWithData(msgs))
}

type linkPreviewReq struct {
	URL string `json:"url" binding:"required"`
}

type linkPreviewResp struct {
	URL         string `json:"url"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Image       string `json:"image"`
	SiteName    string `json:"site_name"`
}

var (
	reTitle       = regexp.MustCompile(`(?i)<title[^>]*>([^<]+)</title>`)
	reOGTitle     = regexp.MustCompile(`(?i)<meta[^>]+property=["']og:title["'][^>]+content=["']([^"']+)["']`)
	reOGDesc      = regexp.MustCompile(`(?i)<meta[^>]+property=["']og:description["'][^>]+content=["']([^"']+)["']`)
	reOGImage     = regexp.MustCompile(`(?i)<meta[^>]+property=["']og:image["'][^>]+content=["']([^"']+)["']`)
	reOGSiteName  = regexp.MustCompile(`(?i)<meta[^>]+property=["']og:site_name["'][^>]+content=["']([^"']+)["']`)
	reMetaDesc    = regexp.MustCompile(`(?i)<meta[^>]+name=["']description["'][^>]+content=["']([^"']+)["']`)
)

func extractMeta(html string) (title, desc, image, site string) {
	if m := reOGTitle.FindStringSubmatch(html); len(m) > 1 {
		title = m[1]
	}
	if title == "" {
		if m := reTitle.FindStringSubmatch(html); len(m) > 1 {
			title = strings.TrimSpace(m[1])
		}
	}
	if m := reOGDesc.FindStringSubmatch(html); len(m) > 1 {
		desc = m[1]
	}
	if desc == "" {
		if m := reMetaDesc.FindStringSubmatch(html); len(m) > 1 {
			desc = m[1]
		}
	}
	if m := reOGImage.FindStringSubmatch(html); len(m) > 1 {
		image = m[1]
	}
	if m := reOGSiteName.FindStringSubmatch(html); len(m) > 1 {
		site = m[1]
	}
	return
}

func (h *IMHandler) HandleLinkPreview(c *gin.Context) {
	var req linkPreviewReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "参数错误"))
		return
	}

	// SSRF 防护：校验 URL 协议和主机
	parsedURL, err := url.Parse(req.URL)
	if err != nil {
		c.JSON(http.StatusOK, response.OKWithData(linkPreviewResp{URL: req.URL}))
		return
	}
	// 只允许 http/https 协议
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		c.JSON(http.StatusOK, response.OKWithData(linkPreviewResp{URL: req.URL}))
		return
	}
	// 禁止访问内网地址
	host := parsedURL.Hostname()
	if isPrivateIP(host) {
		c.JSON(http.StatusOK, response.OKWithData(linkPreviewResp{URL: req.URL}))
		return
	}

	httpReq, err := http.NewRequest("GET", req.URL, nil)
	if err != nil {
		c.JSON(http.StatusOK, response.OKWithData(linkPreviewResp{URL: req.URL}))
		return
	}
	httpReq.Header.Set("User-Agent", "CloudNexus-LinkPreview/1.0")
	httpReq.Header.Set("Accept", "text/html,application/xhtml+xml")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		c.JSON(http.StatusOK, response.OKWithData(linkPreviewResp{URL: req.URL}))
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 512*1024)) // 512KB max
	if err != nil {
		c.JSON(http.StatusOK, response.OKWithData(linkPreviewResp{URL: req.URL}))
		return
	}

	title, desc, image, site := extractMeta(string(body))
	c.JSON(http.StatusOK, response.OKWithData(linkPreviewResp{
		URL: req.URL, Title: title, Description: desc,
		Image: image, SiteName: site,
	}))
}

func (h *IMHandler) HandleExportConversation(c *gin.Context) {
	userID := c.GetUint64("user_id")
	convID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "无效的会话ID"))
		return
	}
	export, err := h.svc.ExportConversation(userID, convID)
	if err != nil {
		handleError(c, err)
		return
	}
	jsonData, _ := json.MarshalIndent(export, "", "  ")
	filename := fmt.Sprintf("chat_%d_%s.json", convID, time.Now().Format("20060102_150405"))
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Data(http.StatusOK, "application/json", jsonData)
}

func (h *IMHandler) HandleImportConversation(c *gin.Context) {
	userID := c.GetUint64("user_id")
	file, _, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "请上传文件"))
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "读取文件失败"))
		return
	}

	var export model.ChatExport
	if err := json.Unmarshal(data, &export); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "JSON 格式错误"))
		return
	}

	summary, err := h.svc.ImportConversation(userID, &export)
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OKWithData(summary))
}

func handleError(c *gin.Context, err error) {
	if appErr, ok := err.(*apperrors.AppError); ok {
		c.JSON(appErr.Code, response.Error(appErr.Code, appErr.Message))
		return
	}
	c.JSON(http.StatusInternalServerError, response.Error(500, "服务器内部错误"))
}
