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
	return s.repo.FindConversationsByUserID(userID)
}

func (s *IMService) CreatePrivateConversation(creatorID, targetID uint64) (*model.Conversation, error) {
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
