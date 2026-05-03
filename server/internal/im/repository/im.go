package repository

import (
	"github.com/cloudnexus/server/pkg/model"
	"gorm.io/gorm"
)

type IMRepository struct {
	db *gorm.DB
}

func NewIMRepository(db *gorm.DB) *IMRepository {
	return &IMRepository{db: db}
}

func (r *IMRepository) CreateConversation(conv *model.Conversation) error {
	return r.db.Create(conv).Error
}

func (r *IMRepository) FindConversationsByUserID(userID uint64) ([]model.Conversation, error) {
	var convs []model.Conversation
	subQuery := r.db.Table("conversation_members").Select("conversation_id").Where("user_id = ?", userID)
	err := r.db.Where("id IN (?)", subQuery).Order("updated_at DESC").Find(&convs).Error
	return convs, err
}

func (r *IMRepository) FindConversationByID(id uint64) (*model.Conversation, error) {
	var conv model.Conversation
	err := r.db.First(&conv, id).Error
	return &conv, err
}

func (r *IMRepository) FindPrivateConversation(user1, user2 uint64) (*model.Conversation, error) {
	var conv model.Conversation
	subQuery := r.db.Table("conversation_members").
		Select("conversation_id").
		Where("user_id IN (?, ?)", user1, user2).
		Group("conversation_id").
		Having("COUNT(DISTINCT user_id) = 2")
	err := r.db.Where("type = 'private' AND id IN (?)", subQuery).First(&conv).Error
	return &conv, err
}

func (r *IMRepository) AddMember(member *model.ConversationMember) error {
	return r.db.Create(member).Error
}

func (r *IMRepository) GetMembers(conversationID uint64) ([]model.ConversationMember, error) {
	var members []model.ConversationMember
	err := r.db.Where("conversation_id = ?", conversationID).Find(&members).Error
	return members, err
}

func (r *IMRepository) CreateMessage(msg *model.Message) error {
	return r.db.Create(msg).Error
}

func (r *IMRepository) FindMessages(conversationID uint64, before uint64, limit int) ([]model.Message, error) {
	var msgs []model.Message
	query := r.db.Where("conversation_id = ?", conversationID)
	if before > 0 {
		query = query.Where("id < ?", before)
	}
	err := query.Order("id DESC").Limit(limit).Find(&msgs).Error
	// Reverse to chronological order
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}
	return msgs, err
}

func (r *IMRepository) UpdateLastReadSeq(conversationID, userID, seq uint64) error {
	return r.db.Model(&model.ConversationMember{}).
		Where("conversation_id = ? AND user_id = ?", conversationID, userID).
		Update("last_read_seq", seq).Error
}
