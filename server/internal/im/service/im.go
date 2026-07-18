package service

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/cloudnexus/server/internal/im/repository"
	apperrors "github.com/cloudnexus/server/pkg/errors"
	"github.com/cloudnexus/server/pkg/model"
)

type IMService struct {
	repo *repository.IMRepository
	hub  *Hub
}

func NewIMService(repo *repository.IMRepository, hub *Hub) *IMService {
	svc := &IMService{repo: repo, hub: hub}
	hub.msgHandler = svc.handleWSMessage
	return svc
}

func (s *IMService) GetConversations(userID uint64) ([]model.Conversation, error) {
	convs, err := s.repo.FindConversationsByUserID(userID)
	if err != nil {
		return nil, err
	}
	for i := range convs {
		if convs[i].Type == "private" && convs[i].Name == "" {
			if name, err := s.repo.GetPrivateConvName(convs[i].ID, userID); err == nil {
				convs[i].Name = name
			}
		}
	}
	return convs, nil
}

func (s *IMService) CreatePrivateConversation(creatorID, targetID uint64) (*model.Conversation, error) {
	if _, err := s.repo.FindUserByID(targetID); err != nil {
		return nil, apperrors.NewAppError(404, "目标用户不存在", apperrors.ErrNotFound)
	}

	existing, err := s.repo.FindPrivateConversation(creatorID, targetID)
	if err == nil && existing != nil && existing.ID > 0 {
		return existing, nil
	}

	conv := &model.Conversation{
		Type:      "private",
		CreatorID: creatorID,
	}
	if err := s.repo.CreateConversation(conv); err != nil {
		return nil, apperrors.NewAppError(500, "创建会话失败", err)
	}

	s.repo.AddMember(&model.ConversationMember{
		ConversationID: conv.ID,
		UserID:         creatorID,
		Role:           "owner",
		JoinedAt:       time.Now(),
	})
	s.repo.AddMember(&model.ConversationMember{
		ConversationID: conv.ID,
		UserID:         targetID,
		Role:           "member",
		JoinedAt:       time.Now(),
	})

	return conv, nil
}

// --- Friend methods ---

func (s *IMService) SendFriendRequest(fromUserID uint64, toUsername, message string, expiresIn int) (*model.Friend, error) {
	target, err := s.repo.FindUserByUsername(toUsername)
	if err != nil {
		return nil, apperrors.NewAppError(404, "用户不存在", apperrors.ErrNotFound)
	}
	if target.ID == fromUserID {
		return nil, apperrors.NewAppError(400, "不能添加自己为好友", apperrors.ErrBadRequest)
	}

	existing, _ := s.repo.FindFriendRequest(fromUserID, target.ID)
	if existing != nil && existing.ID > 0 {
		switch existing.Status {
		case "accepted":
			return nil, apperrors.NewAppError(409, "已经是好友", apperrors.ErrConflict)
		case "pending":
			if existing.FriendID == fromUserID {
				s.repo.AcceptFriendRequest(existing.ID, fromUserID)
				s.CreatePrivateConversation(fromUserID, target.ID)
				existing.Status = "accepted"
				return existing, nil
			}
			return nil, apperrors.NewAppError(409, "已发送过好友请求", apperrors.ErrConflict)
		}
	}

	var expiresAt *time.Time
	if expiresIn > 0 {
		t := time.Now().Add(time.Duration(expiresIn) * 24 * time.Hour)
		expiresAt = &t
	}

	if err := s.repo.CreateFriendRequestWithDetail(fromUserID, target.ID, message, expiresAt); err != nil {
		return nil, apperrors.NewAppError(500, "发送请求失败", err)
	}
	return &model.Friend{UserID: fromUserID, FriendID: target.ID, Status: "pending", Message: message, ExpiresAt: expiresAt}, nil
}

func (s *IMService) AcceptFriendRequest(requestID, userID uint64) (*model.Conversation, error) {
	if err := s.repo.AcceptFriendRequest(requestID, userID); err != nil {
		return nil, apperrors.NewAppError(404, "请求不存在或已处理", apperrors.ErrNotFound)
	}
	req, err := s.repo.FindFriendByID(requestID)
	if err != nil {
		return nil, apperrors.NewAppError(500, "获取请求信息失败", err)
	}
	return s.CreatePrivateConversation(userID, req.UserID)
}

func (s *IMService) RejectFriendRequest(requestID, userID uint64) error {
	if err := s.repo.RejectFriendRequest(requestID, userID); err != nil {
		return apperrors.NewAppError(404, "请求不存在或已处理", apperrors.ErrNotFound)
	}
	return nil
}

func (s *IMService) ListFriends(userID uint64) ([]model.FriendInfo, error) {
	return s.repo.ListFriends(userID)
}

func (s *IMService) ListPendingRequests(userID uint64) ([]model.FriendInfo, error) {
	return s.repo.ListPendingRequests(userID)
}

func (s *IMService) RemoveFriend(userID, friendID uint64) error {
	return s.repo.RemoveFriend(userID, friendID)
}

func (s *IMService) SetFriendRemark(userID, friendID uint64, remark string) error {
	return s.repo.SetFriendRemark(userID, friendID, remark)
}

func (s *IMService) DeleteConversation(userID, convID uint64) error {
	_, err := s.repo.FindConversationByID(convID)
	if err != nil {
		return apperrors.NewAppError(404, "会话不存在", apperrors.ErrNotFound)
	}

	members, err := s.repo.GetMembers(convID)
	if err != nil {
		return apperrors.NewAppError(500, "查询成员失败", err)
	}

	isMember := false
	for _, m := range members {
		if m.UserID == userID && m.DeletedAt == nil {
			isMember = true
			break
		}
	}
	if !isMember {
		return apperrors.NewAppError(403, "你不是该会话的成员", apperrors.ErrForbidden)
	}

	return s.repo.DeleteConversationForUser(convID, userID)
}

func (s *IMService) GetMessages(conversationID, userID uint64, before uint64, limit int) ([]model.Message, error) {
	isMember, err := s.repo.IsConversationMember(conversationID, userID)
	if err != nil {
		return nil, apperrors.NewAppError(500, "查询会话成员失败", err)
	}
	if !isMember {
		return nil, apperrors.NewAppError(403, "你不是该会话的成员", apperrors.ErrForbidden)
	}
	if limit < 1 || limit > 100 {
		limit = 50
	}
	return s.repo.FindMessages(conversationID, before, limit)
}

func (s *IMService) SearchMessages(userID uint64, keyword string, conversationID uint64, page, pageSize int) ([]model.MessageSearchResult, int64, error) {
	if strings.TrimSpace(keyword) == "" {
		return []model.MessageSearchResult{}, 0, nil
	}
	if conversationID > 0 {
		isMember, err := s.repo.IsConversationMember(conversationID, userID)
		if err != nil {
			return nil, 0, apperrors.NewAppError(500, "查询会话成员失败", err)
		}
		if !isMember {
			return nil, 0, apperrors.NewAppError(403, "你不是该会话的成员", apperrors.ErrForbidden)
		}
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return s.repo.SearchMessages(userID, keyword, conversationID, page, pageSize)
}

func (s *IMService) GetMessageContext(conversationID, messageID, userID uint64, before, after int) ([]model.Message, error) {
	isMember, err := s.repo.IsConversationMember(conversationID, userID)
	if err != nil {
		return nil, apperrors.NewAppError(500, "查询会话成员失败", err)
	}
	if !isMember {
		return nil, apperrors.NewAppError(403, "你不是该会话的成员", apperrors.ErrForbidden)
	}
	msg, err := s.repo.FindMessageByID(messageID)
	if err != nil {
		return nil, apperrors.NewAppError(404, "消息不存在", apperrors.ErrNotFound)
	}
	if msg.ConversationID != conversationID {
		return nil, apperrors.NewAppError(400, "消息不属于该会话", apperrors.ErrBadRequest)
	}
	if before < 0 {
		before = 20
	}
	if after < 0 {
		after = 20
	}
	if before > 100 {
		before = 100
	}
	if after > 100 {
		after = 100
	}
	return s.repo.FindMessageContext(conversationID, msg.Seq, before, after)
}

// --- Group chat methods ---

func (s *IMService) CreateGroupConversation(creatorID uint64, name string, memberIDs []uint64) (*model.Conversation, error) {
	if strings.TrimSpace(name) == "" {
		return nil, apperrors.NewAppError(400, "群名称不能为空", apperrors.ErrBadRequest)
	}

	if len(memberIDs) == 0 {
		return nil, apperrors.NewAppError(400, "请至少添加一位成员", apperrors.ErrBadRequest)
	}

	conv := &model.Conversation{
		Type:      "group",
		Name:      name,
		CreatorID: creatorID,
	}
	if err := s.repo.CreateConversation(conv); err != nil {
		return nil, apperrors.NewAppError(500, "创建群聊失败", err)
	}

	s.repo.AddMember(&model.ConversationMember{
		ConversationID: conv.ID,
		UserID:         creatorID,
		Role:           "owner",
		JoinedAt:       time.Now(),
	})

	for _, uid := range memberIDs {
		if uid == creatorID {
			continue
		}
		s.repo.AddMember(&model.ConversationMember{
			ConversationID: conv.ID,
			UserID:         uid,
			Role:           "member",
			JoinedAt:       time.Now(),
		})
	}

	s.sendSystemMessage(conv.ID, conv.LastMsgSeq+1, fmt.Sprintf("%s 创建了群聊", "群主"))
	conv.LastMsgSeq++
	s.repo.CreateConversation(conv)

	return conv, nil
}

func (s *IMService) GetGroupMembers(convID uint64, userID uint64) ([]model.ConversationMember, error) {
	conv, err := s.repo.FindConversationByID(convID)
	if err != nil {
		return nil, apperrors.NewAppError(404, "会话不存在", apperrors.ErrNotFound)
	}
	if conv.Type != "group" {
		return nil, apperrors.NewAppError(400, "非群聊会话", apperrors.ErrBadRequest)
	}

	members, err := s.repo.GetActiveMembers(convID)
	if err != nil {
		return nil, apperrors.NewAppError(500, "获取成员失败", err)
	}

	return members, nil
}

func (s *IMService) AddGroupMember(operatorID uint64, convID uint64, userID uint64) error {
	members, err := s.repo.GetActiveMembers(convID)
	if err != nil {
		return apperrors.NewAppError(500, "查询成员失败", err)
	}

	isMember := false
	for _, m := range members {
		if m.UserID == operatorID {
			isMember = true
			break
		}
	}
	if !isMember {
		return apperrors.NewAppError(403, "你不是群成员", apperrors.ErrForbidden)
	}

	for _, m := range members {
		if m.UserID == userID {
			return apperrors.NewAppError(409, "用户已在群中", apperrors.ErrConflict)
		}
	}

	s.repo.AddMember(&model.ConversationMember{
		ConversationID: convID,
		UserID:         userID,
		Role:           "member",
		JoinedAt:       time.Now(),
	})

	return nil
}

func (s *IMService) RemoveGroupMember(operatorID uint64, convID uint64, targetID uint64) error {
	members, err := s.repo.GetActiveMembers(convID)
	if err != nil {
		return apperrors.NewAppError(500, "查询成员失败", err)
	}

	isMember := false
	for _, m := range members {
		if m.UserID == operatorID {
			isMember = true
			break
		}
	}
	if !isMember {
		return apperrors.NewAppError(403, "你不是群成员", apperrors.ErrForbidden)
	}

	if operatorID == targetID {
		return apperrors.NewAppError(400, "请使用退出群聊", apperrors.ErrBadRequest)
	}

	targetInGroup := false
	for _, m := range members {
		if m.UserID == targetID {
			targetInGroup = true
			break
		}
	}
	if !targetInGroup {
		return apperrors.NewAppError(404, "用户不在群中", apperrors.ErrNotFound)
	}

	return s.repo.DeleteConversationForUser(convID, targetID)
}

func (s *IMService) LeaveGroup(userID uint64, convID uint64) error {
	return s.repo.DeleteConversationForUser(convID, userID)
}

func (s *IMService) sendSystemMessage(convID uint64, seq int64, content string) {
	msg := &model.Message{
		ConversationID: convID,
		SenderID:       0,
		Content:        content,
		MsgType:        "system",
		Seq:            seq,
		CreatedAt:      time.Now(),
	}
	s.repo.CreateMessage(msg)
}

func (s *IMService) handleWSMessage(msg *WSMessage) {
	switch msg.Type {
	case "message":
		s.handleChatMessage(msg)
	case "ping":
		s.hub.SendToUser(msg.SenderID, WSMessage{Type: "pong"})
	case "read_receipt":
		s.handleReadReceipt(msg)
	}
}

func (s *IMService) handleReadReceipt(msg *WSMessage) {
	s.repo.UpdateLastReadSeq(msg.ConversationID, msg.SenderID, msg.LastReadMsgID)

	members, err := s.repo.GetMembers(msg.ConversationID)
	if err != nil {
		return
	}
	for _, member := range members {
		if member.UserID != msg.SenderID {
			s.hub.SendToUser(member.UserID, WSMessage{
				Type:           "read_receipt",
				ConversationID: msg.ConversationID,
				SenderID:       msg.SenderID,
				LastReadMsgID:  msg.LastReadMsgID,
			})
		}
	}
}

func (s *IMService) GetUnreadCounts(userID uint64) map[uint64]int64 {
	return s.repo.GetUnreadCounts(userID)
}

func (s *IMService) GetLastMessages(userID uint64) map[uint64]repository.LastMessageInfo {
	return s.repo.GetLastMessages(userID)
}

func (s *IMService) handleChatMessage(msg *WSMessage) {
	conv, err := s.repo.FindConversationByID(msg.ConversationID)
	if err != nil {
		s.hub.SendToUser(msg.SenderID, WSMessage{
			Type:    "error",
			Content: "会话不存在",
		})
		return
	}

	members, err := s.repo.GetMembers(msg.ConversationID)
	if err != nil {
		return
	}

	newMsg := &model.Message{
		ConversationID: msg.ConversationID,
		SenderID:       msg.SenderID,
		Content:        msg.Content,
		MsgType:        msg.MsgType,
		Seq:            conv.LastMsgSeq + 1,
		CreatedAt:      time.Now(),
	}
	if newMsg.MsgType == "" {
		newMsg.MsgType = "text"
	}

	if err := s.repo.CreateMessage(newMsg); err != nil {
		s.hub.SendToUser(msg.SenderID, WSMessage{
			Type:    "error",
			Content: "消息发送失败",
		})
		return
	}

	s.repo.UpdateConversationSeq(msg.ConversationID, newMsg.Seq)

	outMsg := WSMessage{
		Type:           "message",
		ID:             newMsg.ID,
		ConversationID: newMsg.ConversationID,
		SenderID:       newMsg.SenderID,
		Content:        newMsg.Content,
		MsgType:        newMsg.MsgType,
		CreatedAt:      newMsg.CreatedAt.Format(time.RFC3339),
	}

	for _, member := range members {
		s.hub.SendToUser(member.UserID, outMsg)
	}

	ackData, _ := json.Marshal(WSMessage{
		Type:   "ack",
		MsgID:  newMsg.ID,
		Status: "delivered",
	})
	s.hub.SendToUser(msg.SenderID, WSMessage{
		Type:   "ack",
		MsgID:  newMsg.ID,
		Status: "delivered",
	})
	_ = ackData
}

func (s *IMService) ExportConversation(userID uint64, conversationID uint64) (*model.ChatExport, error) {
	conv, err := s.repo.FindConversationByID(conversationID)
	if err != nil {
		return nil, apperrors.NewAppError(404, "会话不存在", apperrors.ErrNotFound)
	}

	members, err := s.repo.GetActiveMembers(conversationID)
	if err != nil {
		return nil, apperrors.NewAppError(500, "查询成员失败", err)
	}

	isMember := false
	for _, m := range members {
		if m.UserID == userID {
			isMember = true
			break
		}
	}
	if !isMember {
		return nil, apperrors.NewAppError(403, "你不是该会话的成员", apperrors.ErrForbidden)
	}

	msgs, err := s.repo.FindAllMessages(conversationID)
	if err != nil {
		return nil, apperrors.NewAppError(500, "查询消息失败", err)
	}

	participants := make([]string, 0, len(members))
	for _, m := range members {
		u, err := s.repo.FindUserByID(m.UserID)
		if err == nil {
			participants = append(participants, u.Username)
		}
	}

	currentUser, _ := s.repo.FindUserByID(userID)
	exportedBy := ""
	if currentUser != nil {
		exportedBy = currentUser.Username
	}

	convName := conv.Name
	if conv.Type == "private" && convName == "" {
		if name, err := s.repo.GetPrivateConvName(conversationID, userID); err == nil {
			convName = name
		}
	}

	lastSeq := int64(0)
	if len(msgs) > 0 {
		lastSeq = msgs[len(msgs)-1].Seq
	}

	checksum := computeChecksum(conversationID, len(msgs), lastSeq)

	return &model.ChatExport{
		Version:          "1.0",
		ConversationID:   conversationID,
		ConversationType: conv.Type,
		ConversationName: convName,
		Participants:     participants,
		ExportedAt:       time.Now(),
		ExportedBy:       exportedBy,
		MessageCount:     len(msgs),
		LastMessageSeq:   lastSeq,
		Checksum:         checksum,
		Messages:         msgs,
	}, nil
}

func (s *IMService) ImportConversation(userID uint64, export *model.ChatExport) (*model.ImportSummary, error) {
	expected := computeChecksum(export.ConversationID, export.MessageCount, export.LastMessageSeq)
	if export.Checksum != expected {
		return nil, apperrors.NewAppError(400, "校验码不匹配，文件可能已损坏", apperrors.ErrBadRequest)
	}

	members, err := s.repo.GetActiveMembers(export.ConversationID)
	if err != nil {
		return nil, apperrors.NewAppError(500, "查询成员失败", err)
	}

	isMember := false
	for _, m := range members {
		if m.UserID == userID {
			isMember = true
			break
		}
	}
	if !isMember {
		return nil, apperrors.NewAppError(403, "你不是该会话的成员", apperrors.ErrForbidden)
	}

	ids := make([]uint64, 0, len(export.Messages))
	for _, m := range export.Messages {
		ids = append(ids, m.ID)
	}
	existing, err := s.repo.FindExistingMessageIDs(ids)
	if err != nil {
		return nil, apperrors.NewAppError(500, "查询已有消息失败", err)
	}

	newMsgs := make([]model.Message, 0)
	for _, m := range export.Messages {
		if existing[m.ID] {
			continue
		}
		newMsgs = append(newMsgs, model.Message{
			ID:             m.ID,
			ConversationID: m.ConversationID,
			SenderID:       m.SenderID,
			Content:        m.Content,
			MsgType:        m.MsgType,
			Seq:            m.Seq,
			CreatedAt:      m.CreatedAt,
		})
	}

	if err := s.repo.BatchCreateMessages(newMsgs); err != nil {
		return nil, apperrors.NewAppError(500, "导入消息失败", err)
	}

	return &model.ImportSummary{
		Inserted: len(newMsgs),
		Skipped:  len(export.Messages) - len(newMsgs),
		Total:    len(export.Messages),
		LastSeq:  export.LastMessageSeq,
	}, nil
}

func computeChecksum(convID uint64, count int, lastSeq int64) string {
	input := fmt.Sprintf("%d|%d|%d", convID, count, lastSeq)
	sum := sha256.Sum256([]byte(input))
	return fmt.Sprintf("%x", sum)
}
