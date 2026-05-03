package model

import "time"

// User status constants
const (
	UserStatusDisabled = 0
	UserStatusActive   = 1
)

// User represents the users table.
type User struct {
	ID           int64     `gorm:"primaryKey;autoIncrement"`
	Username     string    `gorm:"type:varchar(64);uniqueIndex;not null"`
	Email        string    `gorm:"type:varchar(128);uniqueIndex;not null"`
	PasswordHash string    `gorm:"type:varchar(255);not null;column:password_hash"`
	Status       int8      `gorm:"type:tinyint;not null;default:1"`
	CreatedAt    time.Time `gorm:"not null"`
	UpdatedAt    time.Time `gorm:"not null"`
}

// TableName overrides the default table name.
func (User) TableName() string {
	return "users"
}
