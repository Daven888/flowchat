package model

import "time"

// Message role constants
const (
	MessageRoleUser      = "user"
	MessageRoleAssistant = "assistant"
	MessageRoleSystem    = "system"
)

// Message status constants
const (
	MessageStatusGenerating = "generating"
	MessageStatusCompleted  = "completed"
	MessageStatusFailed     = "failed"
	MessageStatusCancelled  = "cancelled"
)

// ChatMessage represents the chat_messages table.
type ChatMessage struct {
	ID           int64     `gorm:"primaryKey;autoIncrement;index:idx_session_id_id,priority:2"`
	SessionID    int64     `gorm:"not null;index:idx_session_id_id,priority:1"`
	UserID       int64     `gorm:"not null;index:idx_user_created,priority:1"`
	Role         string    `gorm:"type:varchar(32);not null"`
	Content      string    `gorm:"type:mediumtext;not null"`
	Status       string    `gorm:"type:varchar(32);not null;default:completed"`
	ErrorMessage *string   `gorm:"type:varchar(255);column:error_message"`
	TokenCount   int       `gorm:"not null;default:0;column:token_count"`
	CreatedAt    time.Time `gorm:"not null;index:idx_user_created,priority:2"`
	UpdatedAt    time.Time `gorm:"not null"`
}

// TableName overrides the default table name.
func (ChatMessage) TableName() string {
	return "chat_messages"
}
