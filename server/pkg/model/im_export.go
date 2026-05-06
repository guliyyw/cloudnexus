package model

import "time"

type ChatExport struct {
	Version          string          `json:"version"`
	ConversationID   uint64          `json:"conversation_id,string"`
	ConversationType string          `json:"conversation_type"`
	ConversationName string          `json:"conversation_name"`
	Participants     []string        `json:"participants"`
	ExportedAt       time.Time       `json:"exported_at"`
	ExportedBy       string          `json:"exported_by"`
	MessageCount     int             `json:"message_count"`
	LastMessageSeq   int64           `json:"last_message_seq"`
	Checksum         string          `json:"checksum"`
	Messages         []ExportMessage `json:"messages"`
}

type ExportMessage struct {
	ID             uint64    `json:"id,string"`
	ConversationID uint64    `json:"conversation_id,string"`
	SenderID       uint64    `json:"sender_id,string"`
	SenderName     string    `json:"sender_name"`
	Content        string    `json:"content"`
	MsgType        string    `json:"msg_type"`
	Seq            int64     `json:"seq"`
	CreatedAt      time.Time `json:"created_at"`
}

type ImportSummary struct {
	Inserted int   `json:"inserted"`
	Skipped  int   `json:"skipped"`
	Total    int   `json:"total"`
	LastSeq  int64 `json:"last_seq"`
}
