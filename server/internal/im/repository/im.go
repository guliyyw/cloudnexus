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
	subQuery := r.db.Table("conversation_members").Select("conversation_id").Where("user_id = ? AND deleted_at IS NULL", userID)
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
		Where("user_id IN (?, ?) AND deleted_at IS NULL", user1, user2).
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

func (r *IMRepository) UpdateConversationSeq(convID uint64, seq int64) error {
	return r.db.Model(&model.Conversation{}).Where("id = ?", convID).Updates(map[string]interface{}{
		"last_msg_seq": seq,
		"updated_at":   gorm.Expr("now()"),
	}).Error
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

func (r *IMRepository) FindUserByID(id uint64) (*model.User, error) {
	var user model.User
	err := r.db.First(&user, id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *IMRepository) DeleteConversationForUser(convID, userID uint64) error {
	return r.db.Model(&model.ConversationMember{}).
		Where("conversation_id = ? AND user_id = ? AND deleted_at IS NULL", convID, userID).
		Update("deleted_at", gorm.Expr("now()")).Error
}

// --- Friend methods ---

func (r *IMRepository) CreateFriendRequest(userID, friendID uint64) error {
	return r.db.Create(&model.Friend{UserID: userID, FriendID: friendID, Status: "pending"}).Error
}

func (r *IMRepository) FindFriendRequest(userID, friendID uint64) (*model.Friend, error) {
	var f model.Friend
	err := r.db.Where(
		"(user_id = ? AND friend_id = ?) OR (user_id = ? AND friend_id = ?)",
		userID, friendID, friendID, userID,
	).First(&f).Error
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func (r *IMRepository) FindFriendByID(id uint64) (*model.Friend, error) {
	var f model.Friend
	err := r.db.First(&f, id).Error
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func (r *IMRepository) AcceptFriendRequest(requestID, userID uint64) error {
	return r.db.Model(&model.Friend{}).
		Where("id = ? AND friend_id = ? AND status = 'pending'", requestID, userID).
		Update("status", "accepted").Error
}

func (r *IMRepository) RejectFriendRequest(requestID, userID uint64) error {
	return r.db.Where("id = ? AND friend_id = ? AND status = 'pending'", requestID, userID).
		Delete(&model.Friend{}).Error
}

func (r *IMRepository) ListFriends(userID uint64) ([]model.FriendInfo, error) {
	var friends []model.FriendInfo
	err := r.db.Raw(`
		SELECT f.id, f.user_id, f.friend_id, f.status, f.created_at, f.updated_at,
			u.username AS friend_username
		FROM friends f
		JOIN users u ON u.id = CASE WHEN f.user_id = ? THEN f.friend_id ELSE f.user_id END
		WHERE (f.user_id = ? OR f.friend_id = ?) AND f.status = 'accepted'
		ORDER BY f.updated_at DESC
	`, userID, userID, userID).Scan(&friends).Error
	return friends, err
}

func (r *IMRepository) ListPendingRequests(userID uint64) ([]model.FriendInfo, error) {
	var requests []model.FriendInfo
	err := r.db.Raw(`
		SELECT f.id, f.user_id, f.friend_id, f.status, f.created_at, f.updated_at,
			u.username AS friend_username
		FROM friends f
		JOIN users u ON u.id = f.user_id
		WHERE f.friend_id = ? AND f.status = 'pending'
		ORDER BY f.created_at DESC
	`, userID).Scan(&requests).Error
	return requests, err
}

func (r *IMRepository) RemoveFriend(userID, friendID uint64) error {
	return r.db.Where(
		"(user_id = ? AND friend_id = ?) OR (user_id = ? AND friend_id = ?)",
		userID, friendID, friendID, userID,
	).Where("status = 'accepted'").Delete(&model.Friend{}).Error
}

func (r *IMRepository) GetActiveMembers(convID uint64) ([]model.ConversationMember, error) {
	var members []model.ConversationMember
	err := r.db.Where("conversation_id = ? AND deleted_at IS NULL", convID).Find(&members).Error
	return members, err
}

func (r *IMRepository) FindUserByUsername(username string) (*model.User, error) {
	var user model.User
	err := r.db.Where("username = ?", username).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *IMRepository) GetUnreadCount(convID uint64, userID uint64) int64 {
	var member model.ConversationMember
	if err := r.db.Where("conversation_id = ? AND user_id = ? AND deleted_at IS NULL", convID, userID).First(&member).Error; err != nil {
		return 0
	}
	var lastSeq int64
	r.db.Model(&model.Conversation{}).Where("id = ?", convID).Select("last_msg_seq").Scan(&lastSeq)
	c := lastSeq - member.LastReadSeq
	if c < 0 {
		return 0
	}
	return c
}

func (r *IMRepository) GetUnreadCounts(userID uint64) map[uint64]int64 {
	var members []model.ConversationMember
	r.db.Where("user_id = ? AND deleted_at IS NULL", userID).Find(&members)
	counts := make(map[uint64]int64)
	for _, m := range members {
		var lastSeq int64
		r.db.Model(&model.Conversation{}).Where("id = ?", m.ConversationID).Select("last_msg_seq").Scan(&lastSeq)
		c := lastSeq - m.LastReadSeq
		if c < 0 {
			c = 0
		}
		counts[m.ConversationID] = c
	}
	return counts
}

func (r *IMRepository) GetPrivateConvName(conversationID, currentUserID uint64) (string, error) {
	var username string
	err := r.db.Table("conversation_members").
		Select("users.username").
		Joins("JOIN users ON users.id = conversation_members.user_id").
		Where("conversation_members.conversation_id = ? AND conversation_members.user_id != ?", conversationID, currentUserID).
		Limit(1).
		Scan(&username).Error
	return username, err
}
