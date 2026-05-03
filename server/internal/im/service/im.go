package service

import (
	"encoding/json"
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
	// 检查目标用户是否存在
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

func (s *IMService) SendFriendRequest(fromUserID uint64, toUsername string) (*model.Friend, error) {
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
				// The other user already sent us a request, auto-accept it
				s.repo.AcceptFriendRequest(existing.ID, fromUserID)
				s.CreatePrivateConversation(fromUserID, target.ID)
				existing.Status = "accepted"
				return existing, nil
			}
			return nil, apperrors.NewAppError(409, "已发送过好友请求", apperrors.ErrConflict)
		}
	}

	if err := s.repo.CreateFriendRequest(fromUserID, target.ID); err != nil {
		return nil, apperrors.NewAppError(500, "发送请求失败", err)
	}
	return &model.Friend{UserID: fromUserID, FriendID: target.ID, Status: "pending"}, nil
}

func (s *IMService) AcceptFriendRequest(requestID, userID uint64) (*model.Conversation, error) {
	if err := s.repo.AcceptFriendRequest(requestID, userID); err != nil {
		return nil, apperrors.NewAppError(404, "请求不存在或已处理", apperrors.ErrNotFound)
	}
	req, err := s.repo.FindFriendByID(requestID)
	if err != nil {
		return nil, apperrors.NewAppError(500, "获取请求信息失败", err)
	}
	// Auto-create private conversation
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
	if limit < 1 || limit > 100 {
		limit = 50
	}
	return s.repo.FindMessages(conversationID, before, limit)
}

func (s *IMService) handleWSMessage(msg *WSMessage) {
	switch msg.Type {
	case "message":
		s.handleChatMessage(msg)
	case "ping":
		s.hub.SendToUser(msg.SenderID, WSMessage{Type: "pong"})
	case "read_receipt":
		s.repo.UpdateLastReadSeq(msg.ConversationID, msg.SenderID, msg.LastReadMsgID)
	}
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

	// Update conversation
	conv.LastMsgSeq = newMsg.Seq
	conv.UpdatedAt = time.Now()

	outMsg := WSMessage{
		Type:           "message",
		ID:             newMsg.ID,
		ConversationID: newMsg.ConversationID,
		SenderID:       newMsg.SenderID,
		Content:        newMsg.Content,
		MsgType:        newMsg.MsgType,
		CreatedAt:      newMsg.CreatedAt.Format(time.RFC3339),
	}

	// Send to all members of the conversation
	for _, member := range members {
		s.hub.SendToUser(member.UserID, outMsg)
	}

	// Ack to sender
	ackData, _ := json.Marshal(WSMessage{
		Type:  "ack",
		MsgID: newMsg.ID,
		Status: "delivered",
	})
	s.hub.SendToUser(msg.SenderID, WSMessage{
		Type:  "ack",
		MsgID: newMsg.ID,
		Status: "delivered",
	})
	_ = ackData
}
