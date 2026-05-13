package model

import "time"

// ChatSummary represents a compressed summary of early conversation history.
// It is stored in the chat_session_summaries table, separate from chat_messages,
// so that it does not appear in message lists, search results, or exports.
type ChatSummary struct {
	ID            int64     `gorm:"primaryKey;autoIncrement"`
	SessionID     int64     `gorm:"uniqueIndex;not null"`
	Content       string    `gorm:"type:mediumtext;not null"`
	LastMessageID int64     `gorm:"not null"`
	CreatedAt     time.Time `gorm:"not null"`
	UpdatedAt     time.Time `gorm:"not null"`
}

// TableName overrides the default table name.
func (ChatSummary) TableName() string {
	return "chat_session_summaries"
}
