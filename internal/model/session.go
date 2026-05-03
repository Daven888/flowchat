package model

import "time"

// Session status constants
const (
	SessionStatusDeleted = 0
	SessionStatusActive  = 1
)

// ChatSession represents the chat_sessions table.
type ChatSession struct {
	ID        int64     `gorm:"primaryKey;autoIncrement"`
	UserID    int64     `gorm:"not null;index:idx_user_id;index:idx_user_updated,priority:1"`
	Title     string    `gorm:"type:varchar(128);not null"`
	ModelName string    `gorm:"type:varchar(64);not null;column:model_name"`
	Status    int8      `gorm:"type:tinyint;not null;default:1"`
	CreatedAt time.Time `gorm:"not null"`
	UpdatedAt time.Time `gorm:"not null;index:idx_user_updated,priority:2"`
}

// TableName overrides the default table name.
func (ChatSession) TableName() string {
	return "chat_sessions"
}
