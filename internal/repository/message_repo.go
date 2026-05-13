package repository

import (
	"github.com/Daven888/flowchat/internal/model"
	"github.com/Daven888/flowchat/pkg/mysql"
	"gorm.io/gorm"
)

// MessageRepository provides database access for chat message operations.
type MessageRepository struct{}

// NewMessageRepository creates a new MessageRepository.
func NewMessageRepository() *MessageRepository {
	return &MessageRepository{}
}

// Create inserts a new chat message record.
func (r *MessageRepository) Create(msg *model.ChatMessage) error {
	return mysql.DB.Create(msg).Error
}

// FindByID looks up a message by primary key.
func (r *MessageRepository) FindByID(id int64) (*model.ChatMessage, error) {
	var msg model.ChatMessage
	if err := mysql.DB.First(&msg, id).Error; err != nil {
		return nil, err
	}
	return &msg, nil
}

// FindBySessionID returns all messages for a session, ordered by id ASC.
func (r *MessageRepository) FindBySessionID(sessionID int64) ([]model.ChatMessage, error) {
	var messages []model.ChatMessage
	if err := mysql.DB.Where("session_id = ?", sessionID).
		Order("id ASC").
		Find(&messages).Error; err != nil {
		return nil, err
	}
	return messages, nil
}

// Update applies partial updates to a message by primary key.
func (r *MessageRepository) Update(id int64, updates map[string]interface{}) error {
	return mysql.DB.Model(&model.ChatMessage{}).
		Where("id = ?", id).
		Updates(updates).Error
}

// FindCompletedBySessionID returns the last N completed messages for a session,
// queried by id DESC then reversed to ascending order for context assembly.
func (r *MessageRepository) FindCompletedBySessionID(sessionID int64, limit int) ([]model.ChatMessage, error) {
	var messages []model.ChatMessage
	if err := mysql.DB.
		Where("session_id = ? AND status = ?", sessionID, model.MessageStatusCompleted).
		Order("id DESC").
		Limit(limit).
		Find(&messages).Error; err != nil {
		return nil, err
	}
	// Reverse to ascending order
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}
	return messages, nil
}

// FindAllCompletedBySessionID returns all completed messages for a session, ordered by id ASC.
func (r *MessageRepository) FindAllCompletedBySessionID(sessionID int64) ([]model.ChatMessage, error) {
	var messages []model.ChatMessage
	if err := mysql.DB.
		Where("session_id = ? AND status = ?", sessionID, model.MessageStatusCompleted).
		Order("id ASC").
		Find(&messages).Error; err != nil {
		return nil, err
	}
	return messages, nil
}

// FindBySessionIDBefore returns at most limit messages with id < beforeID for a session,
// ordered by id DESC (caller should reverse for ASC display order).
func (r *MessageRepository) FindBySessionIDBefore(sessionID int64, beforeID int64, limit int) ([]model.ChatMessage, error) {
	var messages []model.ChatMessage
	if err := mysql.DB.
		Where("session_id = ? AND id < ?", sessionID, beforeID).
		Order("id DESC").
		Limit(limit).
		Find(&messages).Error; err != nil {
		return nil, err
	}
	return messages, nil
}

// FindBySessionIDLatest returns the latest messages for a session, ordered by id DESC
// (caller should reverse for ASC display order).
func (r *MessageRepository) FindBySessionIDLatest(sessionID int64, limit int) ([]model.ChatMessage, error) {
	var messages []model.ChatMessage
	if err := mysql.DB.
		Where("session_id = ?", sessionID).
		Order("id DESC").
		Limit(limit).
		Find(&messages).Error; err != nil {
		return nil, err
	}
	return messages, nil
}

// CountCompletedBySessionID returns the count of completed messages in a session.
func (r *MessageRepository) CountCompletedBySessionID(sessionID int64) (int64, error) {
	var count int64
	if err := mysql.DB.Model(&model.ChatMessage{}).
		Where("session_id = ? AND status = ?", sessionID, model.MessageStatusCompleted).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// SearchBySessionID searches messages within a session by content LIKE match,
// ordered by id ASC.
func (r *MessageRepository) SearchBySessionID(sessionID int64, query string, limit int) ([]model.ChatMessage, error) {
	var messages []model.ChatMessage
	if err := mysql.DB.
		Where("session_id = ? AND content LIKE ?", sessionID, "%"+query+"%").
		Order("id ASC").
		Limit(limit).
		Find(&messages).Error; err != nil {
		return nil, err
	}
	return messages, nil
}

// CreateUserAndAssistantMessagesTx creates a user message (completed) and an
// assistant placeholder (generating) in a single transaction. Returns both
// created messages. If either insert fails, the transaction is rolled back.
func (r *MessageRepository) CreateUserAndAssistantMessagesTx(userMsg *model.ChatMessage, assistantMsg *model.ChatMessage) error {
	return mysql.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(userMsg).Error; err != nil {
			return err
		}
		if err := tx.Create(assistantMsg).Error; err != nil {
			return err
		}
		return nil
	})
}
