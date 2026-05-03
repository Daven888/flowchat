package handler

import (
	"time"

	"github.com/Daven888/flowchat/internal/model"
)

// UserResponse is the public user representation (no password_hash).
type UserResponse struct {
	ID        int64  `json:"id"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	Status    int8   `json:"status"`
	CreatedAt string `json:"created_at"`
}

// toUserResponse converts a model.User to a safe UserResponse.
func toUserResponse(user *model.User) UserResponse {
	return UserResponse{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		Status:    user.Status,
		CreatedAt: user.CreatedAt.Format(time.RFC3339),
	}
}

// SessionResponse is the public session representation.
type SessionResponse struct {
	ID        int64  `json:"id"`
	UserID    int64  `json:"user_id"`
	Title     string `json:"title"`
	ModelName string `json:"model_name"`
	Status    int8   `json:"status"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// toSessionResponse converts a model.ChatSession to a SessionResponse.
func toSessionResponse(s *model.ChatSession) SessionResponse {
	return SessionResponse{
		ID:        s.ID,
		UserID:    s.UserID,
		Title:     s.Title,
		ModelName: s.ModelName,
		Status:    s.Status,
		CreatedAt: s.CreatedAt.Format(time.RFC3339),
		UpdatedAt: s.UpdatedAt.Format(time.RFC3339),
	}
}

// MessageResponse is the public message representation.
type MessageResponse struct {
	ID           int64  `json:"id"`
	SessionID    int64  `json:"session_id"`
	UserID       int64  `json:"user_id"`
	Role         string `json:"role"`
	Content      string `json:"content"`
	Status       string `json:"status"`
	ErrorMessage string `json:"error_message,omitempty"`
	TokenCount   int    `json:"token_count"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

// toMessageResponse converts a model.ChatMessage to a MessageResponse.
func toMessageResponse(m *model.ChatMessage) MessageResponse {
	errMsg := ""
	if m.ErrorMessage != nil {
		errMsg = *m.ErrorMessage
	}
	return MessageResponse{
		ID:           m.ID,
		SessionID:    m.SessionID,
		UserID:       m.UserID,
		Role:         m.Role,
		Content:      m.Content,
		Status:       m.Status,
		ErrorMessage: errMsg,
		TokenCount:   m.TokenCount,
		CreatedAt:    m.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    m.UpdatedAt.Format(time.RFC3339),
	}
}

// CallLogResponse is the public call log representation.
type CallLogResponse struct {
	ID               int64  `json:"id"`
	RequestID        string `json:"request_id"`
	UserID           int64  `json:"user_id"`
	SessionID        int64  `json:"session_id"`
	Provider         string `json:"provider"`
	ModelName        string `json:"model_name"`
	Status           string `json:"status"`
	PromptTokens     int    `json:"prompt_tokens"`
	CompletionTokens int    `json:"completion_tokens"`
	LatencyMs        int64  `json:"latency_ms"`
	ErrorCode        string `json:"error_code,omitempty"`
	ErrorMessage     string `json:"error_message,omitempty"`
	FinishReason     string `json:"finish_reason,omitempty"`
	StartedAt        string `json:"started_at"`
	FinishedAt       string `json:"finished_at,omitempty"`
	CreatedAt        string `json:"created_at"`
}

// toCallLogResponse converts a model.ModelCallLog to a CallLogResponse.
func toCallLogResponse(log *model.ModelCallLog) CallLogResponse {
	r := CallLogResponse{
		ID:               log.ID,
		RequestID:        log.RequestID,
		UserID:           log.UserID,
		SessionID:        log.SessionID,
		Provider:         log.Provider,
		ModelName:        log.ModelName,
		Status:           log.Status,
		PromptTokens:     log.PromptTokens,
		CompletionTokens: log.CompletionTokens,
		LatencyMs:        log.LatencyMs,
		StartedAt:        log.StartedAt.Format(time.RFC3339),
		CreatedAt:        log.CreatedAt.Format(time.RFC3339),
	}
	if log.ErrorCode != nil {
		r.ErrorCode = *log.ErrorCode
	}
	if log.ErrorMessage != nil {
		r.ErrorMessage = *log.ErrorMessage
	}
	if log.FinishReason != nil {
		r.FinishReason = *log.FinishReason
	}
	if log.FinishedAt != nil {
		r.FinishedAt = log.FinishedAt.Format(time.RFC3339)
	}
	return r
}

// UsageStatResponse is the public usage stat representation.
type UsageStatResponse struct {
	StatDate         string `json:"stat_date"`
	ModelName        string `json:"model_name"`
	TotalCalls       int    `json:"total_calls"`
	SuccessCalls     int    `json:"success_calls"`
	FailedCalls      int    `json:"failed_calls"`
	TimeoutCalls     int    `json:"timeout_calls"`
	CancelledCalls   int    `json:"cancelled_calls"`
	PromptTokens     int64  `json:"prompt_tokens"`
	CompletionTokens int64  `json:"completion_tokens"`
}

// toUsageStatResponse converts a model.UserModelUsageStat to a UsageStatResponse.
func toUsageStatResponse(s *model.UserModelUsageStat) UsageStatResponse {
	return UsageStatResponse{
		StatDate:         s.StatDate.Format("2006-01-02"),
		ModelName:        s.ModelName,
		TotalCalls:       s.TotalCalls,
		SuccessCalls:     s.SuccessCalls,
		FailedCalls:      s.FailedCalls,
		TimeoutCalls:     s.TimeoutCalls,
		CancelledCalls:   s.CancelledCalls,
		PromptTokens:     s.PromptTokens,
		CompletionTokens: s.CompletionTokens,
	}
}
