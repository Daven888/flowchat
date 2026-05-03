package model

import "time"

// Call log status constants
const (
	CallLogStatusSuccess   = "success"
	CallLogStatusFailed    = "failed"
	CallLogStatusTimeout   = "timeout"
	CallLogStatusCancelled = "cancelled"
)

// ModelCallLog represents the model_call_logs table.
type ModelCallLog struct {
	ID               int64      `gorm:"primaryKey;autoIncrement"`
	RequestID        string     `gorm:"type:varchar(128);uniqueIndex:uk_request_id;not null;column:request_id"`
	UserID           int64      `gorm:"not null;index:idx_user_created,priority:1"`
	SessionID        int64      `gorm:"not null;index:idx_session_id"`
	Provider         string     `gorm:"type:varchar(64);not null"`
	ModelName        string     `gorm:"type:varchar(64);not null;column:model_name"`
	Status           string     `gorm:"type:varchar(32);not null"`
	PromptTokens     int        `gorm:"not null;default:0;column:prompt_tokens"`
	CompletionTokens int        `gorm:"not null;default:0;column:completion_tokens"`
	LatencyMs        int64      `gorm:"not null;default:0;column:latency_ms"`
	ErrorCode        *string    `gorm:"type:varchar(64);column:error_code"`
	ErrorMessage     *string    `gorm:"type:varchar(255);column:error_message"`
	FinishReason     *string    `gorm:"type:varchar(64);column:finish_reason"`
	StartedAt        time.Time  `gorm:"not null"`
	FinishedAt       *time.Time `gorm:""`
	CreatedAt        time.Time  `gorm:"not null;index:idx_user_created,priority:2"`
}

// TableName overrides the default table name.
func (ModelCallLog) TableName() string {
	return "model_call_logs"
}
