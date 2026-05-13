package service

import (
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/Daven888/flowchat/internal/model"
	"github.com/Daven888/flowchat/internal/repository"
	"github.com/Daven888/flowchat/pkg/cryptoutil"
)

var (
	ErrCredentialNotFound = errors.New("provider credential not configured")
	ErrMockNoKey          = errors.New("mock provider does not require an api key")
	ErrInvalidProvider    = errors.New("invalid provider name")
	ErrAPIKeyEmpty        = errors.New("api key cannot be empty")
)

// CredentialService manages user API key credentials with encryption.
type CredentialService struct {
	repo          *repository.UserProviderCredentialRepo
	encryptor     *cryptoutil.AESEncryptor
	providerNames []string
}

// CredentialStatus is the public representation of a credential's configuration state.
type CredentialStatus struct {
	ProviderName string `json:"provider_name"`
	Configured   bool   `json:"configured"`
	KeySuffix    string `json:"key_suffix,omitempty"`
	Status       int8   `json:"status"`
	CreatedAt    string `json:"created_at,omitempty"`
	UpdatedAt    string `json:"updated_at,omitempty"`
}

// NewCredentialService creates a new CredentialService.
func NewCredentialService(repo *repository.UserProviderCredentialRepo, encryptor *cryptoutil.AESEncryptor, providerNames []string) *CredentialService {
	return &CredentialService{
		repo:          repo,
		encryptor:     encryptor,
		providerNames: providerNames,
	}
}

// Upsert encrypts and saves an API key for a user+provider. Returns the key suffix for display.
func (s *CredentialService) Upsert(userID int64, providerName, apiKey string) (string, error) {
	if err := s.validateProvider(providerName); err != nil {
		return "", err
	}

	trimmedKey := strings.TrimSpace(apiKey)
	if trimmedKey == "" {
		return "", ErrAPIKeyEmpty
	}

	encrypted, err := s.encryptor.Encrypt(trimmedKey)
	if err != nil {
		return "", err
	}

	suffix := computeKeySuffix(trimmedKey)

	cred := &model.UserProviderCredential{
		UserID:          userID,
		ProviderName:    providerName,
		EncryptedAPIKey: encrypted,
		KeySuffix:       suffix,
	}

	if err := s.repo.Upsert(cred); err != nil {
		return "", err
	}

	return suffix, nil
}

// GetDecrypted returns the plaintext API key for a user+provider.
func (s *CredentialService) GetDecrypted(userID int64, providerName string) (string, error) {
	cred, err := s.repo.FindByUserIDAndProvider(userID, providerName)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", ErrCredentialNotFound
		}
		return "", err
	}

	return s.encryptor.Decrypt(cred.EncryptedAPIKey)
}

// ListStatus returns configuration status for all known providers.
func (s *CredentialService) ListStatus(userID int64) ([]CredentialStatus, error) {
	creds, err := s.repo.FindByUserID(userID)
	if err != nil {
		return nil, err
	}

	credMap := make(map[string]*model.UserProviderCredential, len(creds))
	for i := range creds {
		credMap[creds[i].ProviderName] = &creds[i]
	}

	result := make([]CredentialStatus, 0, len(s.providerNames))
	for _, pn := range s.providerNames {
		if pn == "mock" {
			result = append(result, CredentialStatus{
				ProviderName: "mock",
				Configured:   true,
				Status:       model.CredentialStatusActive,
			})
			continue
		}

		if c, ok := credMap[pn]; ok {
			result = append(result, CredentialStatus{
				ProviderName: pn,
				Configured:   true,
				KeySuffix:    c.KeySuffix,
				Status:       c.Status,
				CreatedAt:    c.CreatedAt.Format(time.RFC3339),
				UpdatedAt:    c.UpdatedAt.Format(time.RFC3339),
			})
		} else {
			result = append(result, CredentialStatus{
				ProviderName: pn,
				Configured:   false,
			})
		}
	}

	return result, nil
}

// Delete removes a credential for a user+provider.
func (s *CredentialService) Delete(userID int64, providerName string) error {
	if err := s.validateProvider(providerName); err != nil {
		return err
	}
	return s.repo.Delete(userID, providerName)
}

// ValidateProvider checks that the provider name exists in the known list.
func (s *CredentialService) ValidateProvider(providerName string) error {
	return s.validateProvider(providerName)
}

func (s *CredentialService) validateProvider(providerName string) error {
	if providerName == "mock" {
		return ErrMockNoKey
	}
	for _, pn := range s.providerNames {
		if pn == providerName {
			return nil
		}
	}
	return ErrInvalidProvider
}

// computeKeySuffix returns the last 4 characters of an API key for display.
func computeKeySuffix(apiKey string) string {
	if len(apiKey) <= 4 {
		return apiKey
	}
	return apiKey[len(apiKey)-4:]
}
