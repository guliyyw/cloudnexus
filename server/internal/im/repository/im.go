package repository

import (
	"fmt"
	"strings"
	"time"

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

func (r *IMRepository) FindMessageByID(id uint64) (*model.Message, error) {
	var msg model.Message
	err := r.db.Where("id = ?", id).First(&msg).Error
	if err != nil {
		return nil, err
	}
	return &msg, nil
}

func (r *IMRepository) IsConversationMember(conversationID, userID uint64) (bool, error) {
	var count int64
	err := r.db.Model(&model.ConversationMember{}).
		Where("conversation_id = ? AND user_id = ? AND deleted_at IS NULL", conversationID, userID).
		Count(&count).Error
	return count > 0, err
}

func (r *IMRepository) FindMessageContext(conversationID uint64, seq int64, before, after int) ([]model.Message, error) {
	var msgs []model.Message
	startSeq := seq - int64(before)
	if startSeq < 1 {
		startSeq = 1
	}
	endSeq := seq + int64(after)
	err := r.db.Where("conversation_id = ? AND seq BETWEEN ? AND ?", conversationID, startSeq, endSeq).
		Order("seq ASC").
		Find(&msgs).Error
	return msgs, err
}

func (r *IMRepository) SearchMessages(userID uint64, keyword string, conversationID uint64, page, pageSize int) ([]model.MessageSearchResult, int64, error) {
	var results []model.MessageSearchResult
	var total int64

	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return results, 0, nil
	}

	base := r.db.Table("messages m").
		Joins("JOIN conversation_members cm ON cm.conversation_id = m.conversation_id AND cm.user_id = ? AND cm.deleted_at IS NULL", userID).
		Joins("JOIN conversations c ON c.id = m.conversation_id").
		Joins("LEFT JOIN users u ON u.id = m.sender_id").
		Where("m.msg_type = ?", "text").
		Where("m.content ILIKE ?", "%"+keyword+"%")

	if conversationID > 0 {
		base = base.Where("m.conversation_id = ?", conversationID)
	}

	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	query := base.Select(strings.Join([]string{
		"m.id",
		"m.conversation_id",
		"COALESCE(NULLIF(c.name, ''), CASE WHEN c.type = 'private' THEN (",
		"  SELECT u2.username",
		"  FROM conversation_members cm2",
		"  JOIN users u2 ON u2.id = cm2.user_id",
		"  WHERE cm2.conversation_id = m.conversation_id AND cm2.user_id <> ?",
		"  ORDER BY cm2.joined_at ASC",
		"  LIMIT 1",
		") ELSE '' END) AS conversation_name",
		"c.type AS conversation_type",
		"m.sender_id",
		"COALESCE(u.username, '') AS sender_name",
		"m.content",
		"m.msg_type",
		"m.seq",
		"m.created_at",
	}, " "), userID).
		Order("m.created_at DESC").
		Offset(offset).
		Limit(pageSize)

	err := query.Scan(&results).Error
	if err != nil {
		return nil, 0, fmt.Errorf("search messages failed: %w", err)
	}

	return results, total, nil
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

func (r *IMRepository) CreateFriendRequestWithDetail(userID, friendID uint64, message string, expiresAt *time.Time) error {
	return r.db.Create(&model.Friend{
		UserID:    userID,
		FriendID:  friendID,
		Status:    "pending",
		Message:   message,
		ExpiresAt: expiresAt,
	}).Error
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

func (r *IMRepository) SetFriendRemark(userID, friendID uint64, remark string) error {
	return r.db.Model(&model.Friend{}).
		Where("user_id = ? AND friend_id = ? AND status = 'accepted'", userID, friendID).
		Update("remark", remark).Error
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

func (r *IMRepository) FindAllMessages(conversationID uint64) ([]model.ExportMessage, error) {
	var msgs []model.ExportMessage
	err := r.db.Table("messages").
		Select("messages.id, messages.conversation_id, messages.sender_id, users.username AS sender_name, messages.content, messages.msg_type, messages.seq, messages.created_at").
		Joins("LEFT JOIN users ON users.id = messages.sender_id").
		Where("messages.conversation_id = ?", conversationID).
		Order("messages.seq ASC").
		Find(&msgs).Error
	return msgs, err
}

func (r *IMRepository) BatchCreateMessages(msgs []model.Message) error {
	if len(msgs) == 0 {
		return nil
	}
	return r.db.Create(&msgs).Error
}

type LastMessageInfo struct {
	ConversationID uint64
	Content        string
	MsgType        string
}

func (r *IMRepository) GetLastMessages(userID uint64) map[uint64]LastMessageInfo {
	var results []LastMessageInfo
	r.db.Raw(`
		SELECT m.conversation_id, m.content, m.msg_type
		FROM messages m
		INNER JOIN (
			SELECT conversation_id, MAX(seq) as max_seq
			FROM messages
			WHERE conversation_id IN (
				SELECT conversation_id FROM conversation_members WHERE user_id = ? AND deleted_at IS NULL
			)
			GROUP BY conversation_id
		) latest ON m.conversation_id = latest.conversation_id AND m.seq = latest.max_seq
	`, userID).Scan(&results)
	m := make(map[uint64]LastMessageInfo, len(results))
	for _, r := range results {
		m[r.ConversationID] = r
	}
	return m
}

func (r *IMRepository) FindExistingMessageIDs(ids []uint64) (map[uint64]bool, error) {
	if len(ids) == 0 {
		return make(map[uint64]bool), nil
	}
	var existing []uint64
	if err := r.db.Table("messages").Where("id IN ?", ids).Pluck("id", &existing).Error; err != nil {
		return nil, err
	}
	set := make(map[uint64]bool, len(existing))
	for _, id := range existing {
		set[id] = true
	}
	return set, nil
}
