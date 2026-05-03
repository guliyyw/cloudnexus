package model

import "time"

type Conversation struct {
	BaseModel
	Type        string `json:"type" gorm:"not null;size:16"` // private / group
	Name        string `json:"name" gorm:"size:128"`
	CreatorID   uint64 `json:"creator_id" gorm:"not null"`
	LastMsgSeq  int64  `json:"last_msg_seq" gorm:"default:0"`
}

type ConversationMember struct {
	ID             uint64     `json:"id" gorm:"primaryKey;autoIncrement"`
	ConversationID uint64     `json:"conversation_id" gorm:"not null;uniqueIndex:idx_conv_user"`
	UserID         uint64     `json:"user_id" gorm:"not null;uniqueIndex:idx_conv_user"`
	Role           string     `json:"role" gorm:"default:member;size:16"` // owner / admin / member
	LastReadSeq    int64      `json:"last_read_seq" gorm:"default:0"`
	JoinedAt       time.Time  `json:"joined_at"`
	DeletedAt      *time.Time `json:"deleted_at" gorm:"index"`
}

type Friend struct {
	ID        uint64    `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID    uint64    `json:"user_id" gorm:"not null;uniqueIndex:idx_friend_pair"`
	FriendID  uint64    `json:"friend_id" gorm:"not null;uniqueIndex:idx_friend_pair"`
	Status    string    `json:"status" gorm:"not null;size:16;default:pending"` // pending / accepted / blocked
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Message struct {
	ID             uint64    `json:"id" gorm:"primaryKey;autoIncrement"`
	ConversationID uint64    `json:"conversation_id" gorm:"not null;index:idx_messages_conv_seq"`
	SenderID       uint64    `json:"sender_id" gorm:"not null"`
	Content        string    `json:"content" gorm:"not null;type:text"`
	MsgType        string    `json:"msg_type" gorm:"default:text;size:16"` // text / image / file / system
	Seq            int64     `json:"seq" gorm:"not null;index:idx_messages_conv_seq"`
	CreatedAt      time.Time `json:"created_at" gorm:"index"`
}
