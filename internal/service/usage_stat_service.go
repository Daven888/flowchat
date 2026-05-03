package service

import (
	"time"

	"go.uber.org/zap"

	"github.com/Daven888/flowchat/internal/model"
	"github.com/Daven888/flowchat/internal/repository"
	"github.com/Daven888/flowchat/pkg/logger"
)

// UsageStatService handles user model usage statistics.
type UsageStatService struct {
	repo *repository.UsageStatRepository
}

// NewUsageStatService creates a new UsageStatService.
func NewUsageStatService(repo *repository.UsageStatRepository) *UsageStatService {
	return &UsageStatService{repo: repo}
}

// UpdateUsageParams holds the data needed to update usage stats.
type UpdateUsageParams struct {
	UserID           int64
	ModelName        string
	Status           string
	PromptTokens     int
	CompletionTokens int
}

// Update increments today's usage stats for the user/model combination.
// Uses INSERT ... ON DUPLICATE KEY UPDATE for atomic increments.
// Errors are logged via zap but not propagated.
func (s *UsageStatService) Update(params UpdateUsageParams) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	stat := &model.UserModelUsageStat{
		UserID:           params.UserID,
		ModelName:        params.ModelName,
		StatDate:         today,
		TotalCalls:       1,
		PromptTokens:     int64(params.PromptTokens),
		CompletionTokens: int64(params.CompletionTokens),
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	switch params.Status {
	case model.CallLogStatusSuccess:
		stat.SuccessCalls = 1
	case model.CallLogStatusFailed:
		stat.FailedCalls = 1
	case model.CallLogStatusTimeout:
		stat.TimeoutCalls = 1
	case model.CallLogStatusCancelled:
		stat.CancelledCalls = 1
	}

	if err := s.repo.Upsert(stat); err != nil {
		if logger.Log != nil {
			logger.Log.Error("failed to upsert usage stat",
				zap.Error(err),
				zap.Int64("user_id", params.UserID),
				zap.String("model_name", params.ModelName),
			)
		}
	}
}

// List returns usage stats for a user with optional filters.
func (s *UsageStatService) List(userID int64, startDate, endDate time.Time, modelName string) ([]model.UserModelUsageStat, error) {
	return s.repo.FindByUserID(userID, startDate, endDate, modelName)
}
