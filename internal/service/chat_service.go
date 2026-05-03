package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/Daven888/flowchat/internal/lock"
	"github.com/Daven888/flowchat/internal/model"
	"github.com/Daven888/flowchat/internal/provider"
	"github.com/Daven888/flowchat/pkg/logger"
	"github.com/Daven888/flowchat/pkg/uuid"
)

var (
	ErrSessionGenerating = errors.New("session is currently generating")
)

// StreamHandle holds the metadata and channel for an ongoing streaming session.
type StreamHandle struct {
	RequestID          string
	AssistantMessageID int64
	ModelName          string
	ProviderName       string
	StartedAt          time.Time
	Chunks             <-chan provider.ChatChunk
	providerCancel     context.CancelFunc
}

// ChatService orchestrates the streaming chat flow.
type ChatService struct {
	msgSvc      *MessageService
	sessionSvc  *SessionService
	modelSvc    *ModelService
	lockManager *lock.Manager
	lockTTL     int
}

// NewChatService creates a new ChatService.
func NewChatService(msgSvc *MessageService, sessionSvc *SessionService, modelSvc *ModelService, lockMgr *lock.Manager, lockTTL int) *ChatService {
	return &ChatService{
		msgSvc:      msgSvc,
		sessionSvc:  sessionSvc,
		modelSvc:    modelSvc,
		lockManager: lockMgr,
		lockTTL:     lockTTL,
	}
}

// BeginStream validates the request, saves user + placeholder messages,
// loads context, and starts the provider stream.
func (s *ChatService) BeginStream(ctx context.Context, userID, sessionID int64, content string) (*StreamHandle, error) {
	// Validate session
	session, err := s.sessionSvc.Get(userID, sessionID)
	if err != nil {
		return nil, err
	}

	// Look up model config
	modelCfg, err := s.modelSvc.GetModelConfig(session.ModelName)
	if err != nil {
		return nil, err
	}

	// Get provider for this model
	p, err := s.modelSvc.GetProvider(modelCfg)
	if err != nil {
		return nil, err
	}

	// Validate content
	if content == "" {
		return nil, ErrMessageContentEmpty
	}

	// Generate request ID
	requestID := fmt.Sprintf("req_%s", uuid.New()[:8])

	// Acquire session generation lock
	acquired, err := s.lockManager.Acquire(ctx, sessionID, requestID, s.lockTTL)
	if err != nil {
		return nil, fmt.Errorf("lock error: %w", err)
	}
	if !acquired {
		return nil, ErrSessionGenerating
	}

	// On any failure after lock acquisition, release the lock
	lockHeld := true
	var providerCancel context.CancelFunc
	defer func() {
		if lockHeld {
			if providerCancel != nil {
				providerCancel()
			}
			_ = s.lockManager.Release(context.Background(), sessionID, requestID)
		}
	}()

	// Save user message (completed)
	_, err = s.msgSvc.SaveUserMessage(sessionID, userID, content)
	if err != nil {
		return nil, err
	}

	// Create assistant placeholder (generating)
	assistantMsg, err := s.msgSvc.CreateAssistantPlaceholder(sessionID, userID)
	if err != nil {
		return nil, err
	}

	// Load context (last N completed messages from model config)
	contextLimit := modelCfg.MaxContextMessages
	if contextLimit <= 0 {
		contextLimit = 10
	}
	messages, err := s.msgSvc.GetContextMessages(sessionID, userID, contextLimit)
	if err != nil {
		return nil, err
	}

	// Build provider messages
	providerMsgs := make([]provider.ProviderMessage, 0, len(messages))
	for _, m := range messages {
		providerMsgs = append(providerMsgs, provider.ProviderMessage{
			Role:    m.Role,
			Content: m.Content,
		})
	}

	// Create timeout context for provider call
	timeout := time.Duration(modelCfg.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	providerCtx, cancel := context.WithTimeout(ctx, timeout)
	providerCancel = cancel

	// Call provider with retries on initial failure
	startedAt := time.Now()
	maxRetries := modelCfg.MaxRetries
	if maxRetries < 0 {
		maxRetries = 0
	}

	var chunkChan <-chan provider.ChatChunk
	for attempt := 0; attempt <= maxRetries; attempt++ {
		chunkChan, err = p.StreamChat(providerCtx, provider.ChatRequest{
			RequestID: requestID,
			ModelName: modelCfg.APIModel,
			Messages:  providerMsgs,
		})
		if err == nil {
			break
		}
		if attempt < maxRetries {
			if logger.Log != nil {
				logger.Log.Warn("provider call failed, retrying",
					zap.Error(err),
					zap.String("request_id", requestID),
					zap.Int("attempt", attempt+1),
				)
			}
			time.Sleep(time.Duration(attempt+1) * 200 * time.Millisecond)
		}
	}
	if err != nil {
		return nil, err
	}

	// Success: transfer lock ownership to caller (do NOT release on defer)
	lockHeld = false

	return &StreamHandle{
		RequestID:          requestID,
		AssistantMessageID: assistantMsg.ID,
		ModelName:          session.ModelName,
		ProviderName:       modelCfg.Provider,
		StartedAt:          startedAt,
		Chunks:             chunkChan,
		providerCancel:     providerCancel,
	}, nil
}

// Complete marks the assistant message as completed with the generated content.
func (s *ChatService) Complete(sessionID, userID, assistantMsgID int64, fullContent string, tokenCount int) error {
	status := model.MessageStatusCompleted
	_, err := s.msgSvc.UpdateAssistantMessage(assistantMsgID, sessionID, userID, UpdateMessageOptions{
		Content:    &fullContent,
		Status:     &status,
		TokenCount: &tokenCount,
	})
	return err
}

// Fail marks the assistant message as failed.
func (s *ChatService) Fail(sessionID, userID, assistantMsgID int64, errorMsg string) error {
	status := model.MessageStatusFailed
	_, err := s.msgSvc.UpdateAssistantMessage(assistantMsgID, sessionID, userID, UpdateMessageOptions{
		Status:       &status,
		ErrorMessage: &errorMsg,
	})
	return err
}

// Cancel marks the assistant message as cancelled.
func (s *ChatService) Cancel(sessionID, userID, assistantMsgID int64) error {
	status := model.MessageStatusCancelled
	_, err := s.msgSvc.UpdateAssistantMessage(assistantMsgID, sessionID, userID, UpdateMessageOptions{
		Status: &status,
	})
	return err
}

// GetContextMessages returns the last N completed messages for context assembly.
func (s *MessageService) GetContextMessages(sessionID, userID int64, limit int) ([]model.ChatMessage, error) {
	// Validate session
	if _, err := s.sessionSvc.Get(userID, sessionID); err != nil {
		return nil, err
	}
	return s.messageRepo.FindCompletedBySessionID(sessionID, limit)
}

// ReleaseLock releases the session generation lock for the given request.
func (s *ChatService) ReleaseLock(sessionID int64, requestID string) error {
	return s.lockManager.Release(context.Background(), sessionID, requestID)
}

// Cleanup cancels the provider timeout context. Safe to call multiple times.
func (h *StreamHandle) Cleanup() {
	if h.providerCancel != nil {
		h.providerCancel()
	}
}

// AutoGenerateTitle updates the session title based on the first user message,
// but only if the current title is still the default "新的对话".
// Failures are logged via zap and do not affect the chat flow.
func (s *ChatService) AutoGenerateTitle(sessionID, userID int64, content string) {
	title := generateTitle(content)
	if err := s.sessionSvc.UpdateTitleIfDefault(userID, sessionID, title); err != nil {
		if logger.Log != nil {
			logger.Log.Error("failed to auto-generate session title",
				zap.Error(err),
				zap.Int64("session_id", sessionID),
			)
		}
	}
}

const maxTitleRunes = 20

// generateTitle produces a session title from user content.
func generateTitle(content string) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return DefaultSessionTitle
	}
	runes := []rune(content)
	if len(runes) <= maxTitleRunes {
		return content
	}
	return string(runes[:maxTitleRunes]) + "..."
}
