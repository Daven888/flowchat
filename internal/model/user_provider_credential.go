package model

import "time"

// Credential status constants
const (
	CredentialStatusActive  = 1
	CredentialStatusDeleted = 0
)

// UserProviderCredential represents the user_provider_credentials table.
type UserProviderCredential struct {
	ID              int64     `gorm:"primaryKey;autoIncrement"`
	UserID          int64     `gorm:"not null;uniqueIndex:uk_user_provider;index:idx_user_id"`
	ProviderName    string    `gorm:"type:varchar(64);not null;uniqueIndex:uk_user_provider;column:provider_name"`
	EncryptedAPIKey string    `gorm:"type:text;not null;column:encrypted_api_key"`
	KeySuffix       string    `gorm:"type:varchar(8);not null;default:'';column:key_suffix"`
	Status          int8      `gorm:"type:tinyint;not null;default:1"`
	CreatedAt       time.Time `gorm:"not null"`
	UpdatedAt       time.Time `gorm:"not null"`
}

// TableName overrides the default table name.
func (UserProviderCredential) TableName() string {
	return "user_provider_credentials"
}
