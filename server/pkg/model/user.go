package model

import "time"

type User struct {
	BaseModel
	Username          string     `json:"username" gorm:"uniqueIndex;not null;size:64"`
	Email             string     `json:"email" gorm:"uniqueIndex;not null;size:255"`
	Password          string     `json:"-" gorm:"not null;size:255"`
	Avatar            string     `json:"avatar" gorm:"size:512"`
	Nickname          string     `json:"nickname" gorm:"size:50"`
	Phone             string     `json:"phone" gorm:"size:20"`
	EmailVerified     bool       `json:"email_verified" gorm:"default:false"`
	PhoneVerified     bool       `json:"phone_verified" gorm:"default:false"`
	LockedUntil       *time.Time `json:"locked_until,omitempty"`
	LoginFailCount    int        `json:"-" gorm:"default:0"`
	ForceLogoutAfter  *time.Time `json:"force_logout_after,omitempty"`
	DeleteRequestedAt *time.Time `json:"delete_requested_at,omitempty"`
	DeletedAt         *time.Time `json:"-" gorm:"index"`
	Status            int8       `json:"status" gorm:"default:1"`
	IsAdmin           bool       `json:"is_admin" gorm:"default:false"`
	Privacy           string     `json:"privacy" gorm:"type:text;default:'{\"allow_search\":true,\"allow_add_friend\":true,\"show_online\":true}'"`
}

// UserPrivacy holds per-user privacy preferences.
type UserPrivacy struct {
	AllowSearch    bool `json:"allow_search"`
	AllowAddFriend bool `json:"allow_add_friend"`
	ShowOnline     bool `json:"show_online"`
}

// UserBrief is a lightweight user view for search results and friend lists.
type UserBrief struct {
	ID       uint64 `json:"id,string"`
	Username string `json:"username"`
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
	Status   int8   `json:"status"`
}

type RefreshToken struct {
	ID        uint64    `json:"id,string" gorm:"primaryKey"`
	UserID    uint64    `json:"user_id,string" gorm:"not null;index"`
	Token     string    `json:"token" gorm:"uniqueIndex;not null;size:512"`
	ExpiresAt time.Time `json:"expires_at" gorm:"not null"`
	CreatedAt time.Time `json:"created_at"`
}
