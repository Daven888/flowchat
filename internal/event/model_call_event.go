package event

import (
	"encoding/json"
	"time"
)

// ModelCallFinishedEvent is published to Redis Stream after a model call completes.
// It contains all data needed by async side tasks (call log, usage stat, auto title).
// It must NEVER contain user API keys, encrypted credentials, or full assistant replies.
type ModelCallFinishedEvent struct {
	RequestID          string `json:"request_id"`
	UserID             int64  `json:"user_id"`
	SessionID          int64  `json:"session_id"`
	Provider           string `json:"provider"`
	ModelName          string `json:"model_name"`
	Status             string `json:"status"`
	PromptTokens       int    `json:"prompt_tokens"`
	CompletionTokens   int    `json:"completion_tokens"`
	LatencyMs          int64  `json:"latency_ms"`
	ErrorCode          string `json:"error_code,omitempty"`
	ErrorMessage       string `json:"error_message,omitempty"`
	FinishReason       string `json:"finish_reason,omitempty"`
	StartedAt          int64  `json:"started_at"`
	FinishedAt         int64  `json:"finished_at"`
	TitleSourceContent string `json:"title_source_content,omitempty"`
	RetryCount         int    `json:"retry_count"`
}

// DeadLetterEvent wraps a ModelCallFinishedEvent that has exceeded max retries.
type DeadLetterEvent struct {
	OriginalEvent   ModelCallFinishedEvent `json:"original_event"`
	FailedReason    string                 `json:"failed_reason"`
	DeadLetterAt    int64                  `json:"dead_letter_at"`
	FinalRetryCount int                    `json:"final_retry_count"`
}

// Marshal serializes the event to JSON bytes.
func (e *ModelCallFinishedEvent) Marshal() ([]byte, error) {
	return json.Marshal(e)
}

// UnmarshalModelCallEvent deserializes a ModelCallFinishedEvent from JSON bytes.
func UnmarshalModelCallEvent(data []byte) (*ModelCallFinishedEvent, error) {
	var e ModelCallFinishedEvent
	if err := json.Unmarshal(data, &e); err != nil {
		return nil, err
	}
	return &e, nil
}

// MarshalDLQ serializes a dead letter event to JSON bytes.
func (d *DeadLetterEvent) Marshal() ([]byte, error) {
	return json.Marshal(d)
}

// UnmarshalDLQ deserializes a DeadLetterEvent from JSON bytes.
func UnmarshalDLQ(data []byte) (*DeadLetterEvent, error) {
	var d DeadLetterEvent
	if err := json.Unmarshal(data, &d); err != nil {
		return nil, err
	}
	return &d, nil
}

// NewModelCallFinishedEvent creates an event with the current retry count.
func NewModelCallFinishedEvent() *ModelCallFinishedEvent {
	return &ModelCallFinishedEvent{
		StartedAt:  time.Now().UnixMilli(),
		FinishedAt: time.Now().UnixMilli(),
	}
}
