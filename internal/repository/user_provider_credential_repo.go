package repository

import (
	"time"

	"gorm.io/gorm"

	"github.com/Daven888/flowchat/internal/model"
	"github.com/Daven888/flowchat/pkg/mysql"
)

// UserProviderCredentialRepo provides database access for user API key credentials.
type UserProviderCredentialRepo struct{}

// NewUserProviderCredentialRepo creates a new UserProviderCredentialRepo.
func NewUserProviderCredentialRepo() *UserProviderCredentialRepo {
	return &UserProviderCredentialRepo{}
}

// Upsert inserts a new credential or updates the existing one for the same user+provider.
// Returns the saved record (with ID populated on insert).
func (r *UserProviderCredentialRepo) Upsert(cred *model.UserProviderCredential) error {
	var existing model.UserProviderCredential
	err := mysql.DB.Where("user_id = ? AND provider_name = ?", cred.UserID, cred.ProviderName).
		First(&existing).Error

	if err != nil && err != gorm.ErrRecordNotFound {
		return err
	}

	now := time.Now()
	if err == gorm.ErrRecordNotFound {
		// Insert
		cred.CreatedAt = now
		cred.UpdatedAt = now
		cred.Status = model.CredentialStatusActive
		return mysql.DB.Create(cred).Error
	}

	// Update
	updates := map[string]interface{}{
		"encrypted_api_key": cred.EncryptedAPIKey,
		"key_suffix":        cred.KeySuffix,
		"status":            model.CredentialStatusActive,
		"updated_at":        now,
	}
	return mysql.DB.Model(&existing).Updates(updates).Error
}

// FindByUserIDAndProvider looks up a credential by user and provider.
func (r *UserProviderCredentialRepo) FindByUserIDAndProvider(userID int64, providerName string) (*model.UserProviderCredential, error) {
	var cred model.UserProviderCredential
	if err := mysql.DB.Where("user_id = ? AND provider_name = ? AND status = ?",
		userID, providerName, model.CredentialStatusActive).First(&cred).Error; err != nil {
		return nil, err
	}
	return &cred, nil
}

// FindByUserID returns all active credentials for a user.
func (r *UserProviderCredentialRepo) FindByUserID(userID int64) ([]model.UserProviderCredential, error) {
	var creds []model.UserProviderCredential
	if err := mysql.DB.Where("user_id = ? AND status = ?",
		userID, model.CredentialStatusActive).Find(&creds).Error; err != nil {
		return nil, err
	}
	return creds, nil
}

// Delete marks a credential as deleted by setting status to 0.
func (r *UserProviderCredentialRepo) Delete(userID int64, providerName string) error {
	return mysql.DB.Model(&model.UserProviderCredential{}).
		Where("user_id = ? AND provider_name = ?", userID, providerName).
		Update("status", model.CredentialStatusDeleted).Error
}
