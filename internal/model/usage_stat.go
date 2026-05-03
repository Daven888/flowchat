package model

import "time"

// UserModelUsageStat represents the user_model_usage_stats table.
type UserModelUsageStat struct {
	ID               int64     `gorm:"primaryKey;autoIncrement"`
	UserID           int64     `gorm:"not null;uniqueIndex:uk_user_model_date,priority:1;index:idx_user_date,priority:1"`
	ModelName        string    `gorm:"type:varchar(64);not null;uniqueIndex:uk_user_model_date,priority:2;column:model_name"`
	StatDate         time.Time `gorm:"type:date;not null;uniqueIndex:uk_user_model_date,priority:3;index:idx_user_date,priority:2"`
	TotalCalls       int       `gorm:"not null;default:0;column:total_calls"`
	SuccessCalls     int       `gorm:"not null;default:0;column:success_calls"`
	FailedCalls      int       `gorm:"not null;default:0;column:failed_calls"`
	TimeoutCalls     int       `gorm:"not null;default:0;column:timeout_calls"`
	CancelledCalls   int       `gorm:"not null;default:0;column:cancelled_calls"`
	PromptTokens     int64     `gorm:"not null;default:0;column:prompt_tokens"`
	CompletionTokens int64     `gorm:"not null;default:0;column:completion_tokens"`
	CreatedAt        time.Time `gorm:"not null"`
	UpdatedAt        time.Time `gorm:"not null"`
}

// TableName overrides the default table name.
func (UserModelUsageStat) TableName() string {
	return "user_model_usage_stats"
}
