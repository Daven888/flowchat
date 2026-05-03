package service

import (
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/Daven888/flowchat/internal/model"
	"github.com/Daven888/flowchat/internal/repository"
)

var (
	ErrMessageContentEmpty  = errors.New("message content cannot be empty")
	ErrMessageNotFound      = errors.New("message not found")
	ErrInvalidMessageStatus = errors.New("invalid message status")
	ErrNotAssistantMessage  = errors.New("message is not an assistant message")
)

// UpdateMessageOptions holds optional fields for updating an assistant message.
type UpdateMessageOptions struct {
	Content      *string
	Status       *string
	ErrorMessage *string
	TokenCount   *int
}

// MessageService handles chat message business logic.
type MessageService struct {
	messageRepo *repository.MessageRepository
	sessionSvc  *SessionService
}

// NewMessageService creates a new MessageService.
func NewMessageService(messageRepo *repository.MessageRepository, sessionSvc *SessionService) *MessageService {
	return &MessageService{
		messageRepo: messageRepo,
		sessionSvc:  sessionSvc,
	}
}

// SaveUserMessage creates a user message with status=completed.
// This is an internal method called by the streaming chat module.
func (s *MessageService) SaveUserMessage(sessionID, userID int64, content string) (*model.ChatMessage, error) {
	if content == "" {
		return nil, ErrMessageContentEmpty
	}

	// Validate session belongs to user and is active
	if _, err := s.sessionSvc.Get(userID, sessionID); err != nil {
		return nil, err
	}

	now := time.Now()
	msg := &model.ChatMessage{
		SessionID: sessionID,
		UserID:    userID,
		Role:      model.MessageRoleUser,
		Content:   content,
		Status:    model.MessageStatusCompleted,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.messageRepo.Create(msg); err != nil {
		return nil, err
	}

	return msg, nil
}

// CreateAssistantPlaceholder creates an assistant message with status=generating and empty content.
// This is an internal method called by the streaming chat module.
func (s *MessageService) CreateAssistantPlaceholder(sessionID, userID int64) (*model.ChatMessage, error) {
	// Validate session belongs to user and is active
	if _, err := s.sessionSvc.Get(userID, sessionID); err != nil {
		return nil, err
	}

	now := time.Now()
	msg := &model.ChatMessage{
		SessionID: sessionID,
		UserID:    userID,
		Role:      model.MessageRoleAssistant,
		Content:   "",
		Status:    model.MessageStatusGenerating,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.messageRepo.Create(msg); err != nil {
		return nil, err
	}

	return msg, nil
}

// UpdateAssistantMessage updates an assistant message's fields.
// Only allows updating content, status, error_message, and token_count.
// Status must be one of: completed, failed, cancelled.
// This is an internal method called by the streaming chat module.
func (s *MessageService) UpdateAssistantMessage(messageID, sessionID, userID int64, opts UpdateMessageOptions) (*model.ChatMessage, error) {
	// Validate session belongs to user and is active
	if _, err := s.sessionSvc.Get(userID, sessionID); err != nil {
		return nil, err
	}

	msg, err := s.messageRepo.FindByID(messageID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrMessageNotFound
		}
		return nil, err
	}

	// Must belong to the given session
	if msg.SessionID != sessionID {
		return nil, ErrMessageNotFound
	}

	// Must be an assistant message
	if msg.Role != model.MessageRoleAssistant {
		return nil, ErrNotAssistantMessage
	}

	// Validate status if being updated
	if opts.Status != nil {
		switch *opts.Status {
		case model.MessageStatusCompleted, model.MessageStatusFailed, model.MessageStatusCancelled:
			// valid
		default:
			return nil, ErrInvalidMessageStatus
		}
	}

	updates := map[string]interface{}{
		"updated_at": time.Now(),
	}
	if opts.Content != nil {
		updates["content"] = *opts.Content
	}
	if opts.Status != nil {
		updates["status"] = *opts.Status
	}
	if opts.ErrorMessage != nil {
		updates["error_message"] = *opts.ErrorMessage
	}
	if opts.TokenCount != nil {
		updates["token_count"] = *opts.TokenCount
	}

	if err := s.messageRepo.Update(messageID, updates); err != nil {
		return nil, err
	}

	return s.messageRepo.FindByID(messageID)
}

// ListMessages returns all messages for a session, ordered by id ASC.
func (s *MessageService) ListMessages(sessionID, userID int64) ([]model.ChatMessage, error) {
	// Validate session belongs to user and is active
	if _, err := s.sessionSvc.Get(userID, sessionID); err != nil {
		return nil, err
	}

	return s.messageRepo.FindBySessionID(sessionID)
}

// GetAllCompletedMessages returns all completed messages for a session, ordered by id ASC.
func (s *MessageService) GetAllCompletedMessages(sessionID, userID int64) ([]model.ChatMessage, error) {
	if _, err := s.sessionSvc.Get(userID, sessionID); err != nil {
		return nil, err
	}
	return s.messageRepo.FindAllCompletedBySessionID(sessionID)
}
