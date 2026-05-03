package service

import (
	"time"

	"go.uber.org/zap"

	"github.com/Daven888/flowchat/internal/model"
	"github.com/Daven888/flowchat/internal/repository"
	"github.com/Daven888/flowchat/pkg/logger"
)

// CallLogService handles model call log operations.
type CallLogService struct {
	repo *repository.CallLogRepository
}

// NewCallLogService creates a new CallLogService.
func NewCallLogService(repo *repository.CallLogRepository) *CallLogService {
	return &CallLogService{repo: repo}
}

// CreateCallLogParams holds the data needed to create a model call log.
type CreateCallLogParams struct {
	RequestID        string
	UserID           int64
	SessionID        int64
	Provider         string
	ModelName        string
	Status           string
	PromptTokens     int
	CompletionTokens int
	LatencyMs        int64
	ErrorCode        string
	ErrorMessage     string
	FinishReason     string
	StartedAt        time.Time
	FinishedAt       time.Time
}

// Log writes a model call log entry. Errors are logged via zap but not propagated.
func (s *CallLogService) Log(params CreateCallLogParams) {
	var errorCode *string
	if params.ErrorCode != "" {
		errorCode = &params.ErrorCode
	}
	var errorMessage *string
	if params.ErrorMessage != "" {
		errorMessage = &params.ErrorMessage
	}
	var finishReason *string
	if params.FinishReason != "" {
		finishReason = &params.FinishReason
	}

	log := &model.ModelCallLog{
		RequestID:        params.RequestID,
		UserID:           params.UserID,
		SessionID:        params.SessionID,
		Provider:         params.Provider,
		ModelName:        params.ModelName,
		Status:           params.Status,
		PromptTokens:     params.PromptTokens,
		CompletionTokens: params.CompletionTokens,
		LatencyMs:        params.LatencyMs,
		ErrorCode:        errorCode,
		ErrorMessage:     errorMessage,
		FinishReason:     finishReason,
		StartedAt:        params.StartedAt,
		FinishedAt:       &params.FinishedAt,
		CreatedAt:        time.Now(),
	}

	if err := s.repo.Create(log); err != nil {
		if logger.Log != nil {
			logger.Log.Error("failed to write model call log",
				zap.Error(err),
				zap.String("request_id", params.RequestID),
			)
		}
	}
}

// List returns paginated call logs for a user with optional filters.
func (s *CallLogService) List(userID int64, status, modelName string, page, pageSize int) ([]model.ModelCallLog, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return s.repo.FindByUserID(userID, status, modelName, page, pageSize)
}

// Get returns a single call log if it belongs to the user.
func (s *CallLogService) Get(userID, logID int64) (*model.ModelCallLog, error) {
	log, err := s.repo.FindByID(logID)
	if err != nil {
		return nil, err
	}
	if log.UserID != userID {
		return nil, ErrSessionNotFound // reuse generic not-found
	}
	return log, nil
}
