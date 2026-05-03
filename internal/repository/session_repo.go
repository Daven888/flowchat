package repository

import (
	"github.com/Daven888/flowchat/internal/model"
	"github.com/Daven888/flowchat/pkg/mysql"
)

// SessionRepository provides database access for chat session operations.
type SessionRepository struct{}

// NewSessionRepository creates a new SessionRepository.
func NewSessionRepository() *SessionRepository {
	return &SessionRepository{}
}

// Create inserts a new chat session record.
func (r *SessionRepository) Create(session *model.ChatSession) error {
	return mysql.DB.Create(session).Error
}

// FindByID looks up a session by primary key.
func (r *SessionRepository) FindByID(id int64) (*model.ChatSession, error) {
	var session model.ChatSession
	if err := mysql.DB.First(&session, id).Error; err != nil {
		return nil, err
	}
	return &session, nil
}

// FindByUserID returns all active sessions for a user, ordered by updated_at DESC.
func (r *SessionRepository) FindByUserID(userID int64) ([]model.ChatSession, error) {
	var sessions []model.ChatSession
	if err := mysql.DB.
		Where("user_id = ? AND status = ?", userID, model.SessionStatusActive).
		Order("updated_at DESC").
		Find(&sessions).Error; err != nil {
		return nil, err
	}
	return sessions, nil
}

// SoftDelete marks a session as deleted by setting status to 0.
func (r *SessionRepository) SoftDelete(id int64) error {
	return mysql.DB.Model(&model.ChatSession{}).
		Where("id = ?", id).
		Update("status", model.SessionStatusDeleted).Error
}

// UpdateTitle updates the title of a session.
func (r *SessionRepository) UpdateTitle(id int64, title string) error {
	return mysql.DB.Model(&model.ChatSession{}).
		Where("id = ?", id).
		Update("title", title).Error
}
