package service

import (
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/Daven888/flowchat/internal/model"
	"github.com/Daven888/flowchat/internal/repository"
)

var (
	ErrSessionNotFound  = errors.New("session not found")
	ErrInvalidModelName = errors.New("invalid model name")
)

const (
	DefaultSessionTitle = "新的对话"
)

// SessionService handles chat session business logic.
type SessionService struct {
	repo *repository.SessionRepository
}

// NewSessionService creates a new SessionService.
func NewSessionService(repo *repository.SessionRepository) *SessionService {
	return &SessionService{repo: repo}
}

// Create creates a new chat session for the given user.
func (s *SessionService) Create(userID int64, title, modelName string) (*model.ChatSession, error) {
	if modelName == "" {
		return nil, ErrInvalidModelName
	}

	if title == "" {
		title = DefaultSessionTitle
	}

	now := time.Now()
	session := &model.ChatSession{
		UserID:    userID,
		Title:     title,
		ModelName: modelName,
		Status:    model.SessionStatusActive,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.repo.Create(session); err != nil {
		return nil, err
	}

	return session, nil
}

// List returns all active sessions for a user, ordered by updated_at DESC.
func (s *SessionService) List(userID int64) ([]model.ChatSession, error) {
	return s.repo.FindByUserID(userID)
}

// Get returns session detail if it belongs to the user and is active.
func (s *SessionService) Get(userID, sessionID int64) (*model.ChatSession, error) {
	session, err := s.repo.FindByID(sessionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSessionNotFound
		}
		return nil, err
	}

	if session.UserID != userID || session.Status != model.SessionStatusActive {
		return nil, ErrSessionNotFound
	}

	return session, nil
}

// Delete soft-deletes a session if it belongs to the user and is active.
func (s *SessionService) Delete(userID, sessionID int64) error {
	session, err := s.repo.FindByID(sessionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrSessionNotFound
		}
		return err
	}

	if session.UserID != userID || session.Status != model.SessionStatusActive {
		return ErrSessionNotFound
	}

	return s.repo.SoftDelete(sessionID)
}

// UpdateTitleIfDefault updates the session title only if it is currently the default.
// Returns nil (no error) if the title is already customized (not updated).
func (s *SessionService) UpdateTitleIfDefault(userID, sessionID int64, newTitle string) error {
	session, err := s.repo.FindByID(sessionID)
	if err != nil {
		return err
	}
	if session.UserID != userID {
		return ErrSessionNotFound
	}
	if session.Title != DefaultSessionTitle {
		return nil // already customized, skip
	}
	return s.repo.UpdateTitle(sessionID, newTitle)
}
