package event

import (
	"context"
	"fmt"
	"time"

	"github.com/Daven888/flowchat/internal/model"
	"github.com/Daven888/flowchat/internal/repository"
)

// ModelCallEventHandler processes ModelCallFinishedEvent and executes side tasks:
// 1. Write model_call_logs
// 2. Update user_model_usage_stats
// 3. Auto-generate session title (on success)
type ModelCallEventHandler struct {
	callLogRepo *repository.CallLogRepository
	usageRepo   *repository.UsageStatRepository
	sessionRepo *repository.SessionRepository
}

// NewModelCallEventHandler creates a new ModelCallEventHandler.
func NewModelCallEventHandler(
	callLogRepo *repository.CallLogRepository,
	usageRepo *repository.UsageStatRepository,
	sessionRepo *repository.SessionRepository,
) *ModelCallEventHandler {
	return &ModelCallEventHandler{
		callLogRepo: callLogRepo,
		usageRepo:   usageRepo,
		sessionRepo: sessionRepo,
	}
}

// Handle processes a single ModelCallFinishedEvent. Returns error on failure so the
// consumer can retry. All three side tasks are attempted; the first non-nil error is
// returned.
func (h *ModelCallEventHandler) Handle(ctx context.Context, event *ModelCallFinishedEvent) error {
	// 1. Write model call log.
	if err := h.writeCallLog(event); err != nil {
		return fmt.Errorf("write call log: %w", err)
	}

	// 2. Update usage stats.
	if err := h.updateUsageStat(event); err != nil {
		return fmt.Errorf("update usage stat: %w", err)
	}

	// 3. Auto-generate session title on success.
	if event.Status == model.CallLogStatusSuccess && event.TitleSourceContent != "" {
		if err := h.autoGenerateTitle(event); err != nil {
			// Title generation failure is non-critical — log but don't retry.
			// Return nil so the event is ACKed.
		}
	}

	return nil
}

func (h *ModelCallEventHandler) writeCallLog(event *ModelCallFinishedEvent) error {
	var errorCode *string
	if event.ErrorCode != "" {
		errorCode = &event.ErrorCode
	}
	var errorMessage *string
	if event.ErrorMessage != "" {
		errorMessage = &event.ErrorMessage
	}
	var finishReason *string
	if event.FinishReason != "" {
		finishReason = &event.FinishReason
	}

	startedAt := time.UnixMilli(event.StartedAt)
	finishedAt := time.UnixMilli(event.FinishedAt)

	log := &model.ModelCallLog{
		RequestID:        event.RequestID,
		UserID:           event.UserID,
		SessionID:        event.SessionID,
		Provider:         event.Provider,
		ModelName:        event.ModelName,
		Status:           event.Status,
		PromptTokens:     event.PromptTokens,
		CompletionTokens: event.CompletionTokens,
		LatencyMs:        event.LatencyMs,
		ErrorCode:        errorCode,
		ErrorMessage:     errorMessage,
		FinishReason:     finishReason,
		StartedAt:        startedAt,
		FinishedAt:       &finishedAt,
		CreatedAt:        time.Now(),
	}

	return h.callLogRepo.Create(log)
}

func (h *ModelCallEventHandler) updateUsageStat(event *ModelCallFinishedEvent) error {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	stat := &model.UserModelUsageStat{
		UserID:           event.UserID,
		ModelName:        event.ModelName,
		StatDate:         today,
		TotalCalls:       1,
		PromptTokens:     int64(event.PromptTokens),
		CompletionTokens: int64(event.CompletionTokens),
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	switch event.Status {
	case model.CallLogStatusSuccess:
		stat.SuccessCalls = 1
	case model.CallLogStatusFailed:
		stat.FailedCalls = 1
	case model.CallLogStatusTimeout:
		stat.TimeoutCalls = 1
	case model.CallLogStatusCancelled:
		stat.CancelledCalls = 1
	}

	return h.usageRepo.Upsert(stat)
}

func (h *ModelCallEventHandler) autoGenerateTitle(event *ModelCallFinishedEvent) error {
	const maxTitleRunes = 20
	const defaultTitle = "新的对话"

	content := event.TitleSourceContent
	if content == "" {
		return nil
	}

	runes := []rune(content)
	title := content
	if len(runes) > maxTitleRunes {
		title = string(runes[:maxTitleRunes]) + "..."
	}
	if title == "" {
		title = defaultTitle
	}

	// Only update if the current title is still the default.
	session, err := h.sessionRepo.FindByID(event.SessionID)
	if err != nil {
		return fmt.Errorf("find session: %w", err)
	}
	if session.UserID != event.UserID {
		return fmt.Errorf("session %d does not belong to user %d", event.SessionID, event.UserID)
	}
	if session.Title != defaultTitle {
		return nil // already customized, skip
	}

	return h.sessionRepo.UpdateTitle(event.SessionID, title)
}
