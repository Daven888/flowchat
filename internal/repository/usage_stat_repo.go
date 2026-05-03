package repository

import (
	"time"

	"github.com/Daven888/flowchat/internal/model"
	"github.com/Daven888/flowchat/pkg/mysql"
)

// UsageStatRepository provides database access for user model usage stats.
type UsageStatRepository struct{}

// NewUsageStatRepository creates a new UsageStatRepository.
func NewUsageStatRepository() *UsageStatRepository {
	return &UsageStatRepository{}
}

// Upsert inserts a new stat record or updates an existing one atomically
// using INSERT ... ON DUPLICATE KEY UPDATE, relying on the unique index
// uk_user_model_date (user_id, model_name, stat_date).
func (r *UsageStatRepository) Upsert(stat *model.UserModelUsageStat) error {
	sql := `
		INSERT INTO user_model_usage_stats
			(user_id, model_name, stat_date, total_calls, success_calls, failed_calls,
			 timeout_calls, cancelled_calls, prompt_tokens, completion_tokens, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			total_calls = total_calls + VALUES(total_calls),
			success_calls = success_calls + VALUES(success_calls),
			failed_calls = failed_calls + VALUES(failed_calls),
			timeout_calls = timeout_calls + VALUES(timeout_calls),
			cancelled_calls = cancelled_calls + VALUES(cancelled_calls),
			prompt_tokens = prompt_tokens + VALUES(prompt_tokens),
			completion_tokens = completion_tokens + VALUES(completion_tokens),
			updated_at = VALUES(updated_at)
	`
	return mysql.DB.Exec(sql,
		stat.UserID, stat.ModelName, stat.StatDate,
		stat.TotalCalls, stat.SuccessCalls, stat.FailedCalls,
		stat.TimeoutCalls, stat.CancelledCalls, stat.PromptTokens, stat.CompletionTokens,
		stat.CreatedAt, stat.UpdatedAt,
	).Error
}

// FindByUserID returns usage stats for a user with optional filters, ordered by stat_date DESC.
func (r *UsageStatRepository) FindByUserID(userID int64, startDate, endDate time.Time, modelName string) ([]model.UserModelUsageStat, error) {
	query := mysql.DB.Model(&model.UserModelUsageStat{}).Where("user_id = ?", userID)

	if !startDate.IsZero() {
		query = query.Where("stat_date >= ?", startDate)
	}
	if !endDate.IsZero() {
		query = query.Where("stat_date <= ?", endDate)
	}
	if modelName != "" {
		query = query.Where("model_name = ?", modelName)
	}

	var stats []model.UserModelUsageStat
	if err := query.Order("stat_date DESC").Find(&stats).Error; err != nil {
		return nil, err
	}
	return stats, nil
}
