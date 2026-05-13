package event

import (
	"encoding/json"
	"testing"
	"time"
)

func TestModelCallFinishedEventRoundTrip(t *testing.T) {
	original := &ModelCallFinishedEvent{
		RequestID:          "req_abc123",
		UserID:             42,
		SessionID:          100,
		Provider:           "deepseek",
		ModelName:          "deepseek-chat",
		Status:             "success",
		PromptTokens:       150,
		CompletionTokens:   300,
		LatencyMs:          2500,
		ErrorCode:          "",
		ErrorMessage:       "",
		FinishReason:       "stop",
		StartedAt:          time.Now().UnixMilli(),
		FinishedAt:         time.Now().UnixMilli(),
		TitleSourceContent: "你好",
		RetryCount:         0,
	}

	data, err := original.Marshal()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	restored, err := UnmarshalModelCallEvent(data)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if restored.RequestID != original.RequestID {
		t.Errorf("RequestID = %q, want %q", restored.RequestID, original.RequestID)
	}
	if restored.UserID != original.UserID {
		t.Errorf("UserID = %d, want %d", restored.UserID, original.UserID)
	}
	if restored.SessionID != original.SessionID {
		t.Errorf("SessionID = %d, want %d", restored.SessionID, original.SessionID)
	}
	if restored.Provider != original.Provider {
		t.Errorf("Provider = %q, want %q", restored.Provider, original.Provider)
	}
	if restored.ModelName != original.ModelName {
		t.Errorf("ModelName = %q, want %q", restored.ModelName, original.ModelName)
	}
	if restored.Status != original.Status {
		t.Errorf("Status = %q, want %q", restored.Status, original.Status)
	}
	if restored.PromptTokens != original.PromptTokens {
		t.Errorf("PromptTokens = %d, want %d", restored.PromptTokens, original.PromptTokens)
	}
	if restored.CompletionTokens != original.CompletionTokens {
		t.Errorf("CompletionTokens = %d, want %d", restored.CompletionTokens, original.CompletionTokens)
	}
	if restored.LatencyMs != original.LatencyMs {
		t.Errorf("LatencyMs = %d, want %d", restored.LatencyMs, original.LatencyMs)
	}
	if restored.TitleSourceContent != original.TitleSourceContent {
		t.Errorf("TitleSourceContent = %q, want %q", restored.TitleSourceContent, original.TitleSourceContent)
	}
	if restored.RetryCount != original.RetryCount {
		t.Errorf("RetryCount = %d, want %d", restored.RetryCount, original.RetryCount)
	}
}

func TestModelCallFinishedEventNoAPIKeyLeak(t *testing.T) {
	// The event struct must not have any API key or credential fields.
	// Verify by checking that the JSON struct tags don't include any key-like names.
	ev := &ModelCallFinishedEvent{}
	data, _ := ev.Marshal()

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal to map: %v", err)
	}

	forbidden := []string{"api_key", "apikey", "credential", "password", "secret", "token", "encrypted"}
	for _, key := range forbidden {
		if _, ok := raw[key]; ok {
			t.Errorf("event should NOT contain field %q — it may leak sensitive data", key)
		}
	}
}

func TestUnmarshalModelCallEventInvalidJSON(t *testing.T) {
	_, err := UnmarshalModelCallEvent([]byte("not json"))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestUnmarshalModelCallEventEmptyJSON(t *testing.T) {
	ev, err := UnmarshalModelCallEvent([]byte("{}"))
	if err != nil {
		t.Fatalf("unmarshal empty: %v", err)
	}
	if ev.RequestID != "" {
		t.Errorf("RequestID should be empty for empty JSON, got %q", ev.RequestID)
	}
}

func TestDeadLetterEventRoundTrip(t *testing.T) {
	original := &DeadLetterEvent{
		OriginalEvent: ModelCallFinishedEvent{
			RequestID: "req_dlq_test",
			UserID:    1,
			SessionID: 2,
			Status:    "failed",
		},
		FailedReason:    "db connection timeout",
		DeadLetterAt:    time.Now().UnixMilli(),
		FinalRetryCount: 4,
	}

	data, err := original.Marshal()
	if err != nil {
		t.Fatalf("marshal dlq: %v", err)
	}

	restored, err := UnmarshalDLQ(data)
	if err != nil {
		t.Fatalf("unmarshal dlq: %v", err)
	}

	if restored.FailedReason != original.FailedReason {
		t.Errorf("FailedReason = %q, want %q", restored.FailedReason, original.FailedReason)
	}
	if restored.FinalRetryCount != original.FinalRetryCount {
		t.Errorf("FinalRetryCount = %d, want %d", restored.FinalRetryCount, original.FinalRetryCount)
	}
	if restored.OriginalEvent.RequestID != original.OriginalEvent.RequestID {
		t.Errorf("OriginalEvent.RequestID = %q, want %q",
			restored.OriginalEvent.RequestID, original.OriginalEvent.RequestID)
	}
}

func TestEventRetryCountIncrements(t *testing.T) {
	ev := &ModelCallFinishedEvent{RetryCount: 0}
	ev.RetryCount++
	if ev.RetryCount != 1 {
		t.Errorf("RetryCount should be 1, got %d", ev.RetryCount)
	}
	ev.RetryCount++
	if ev.RetryCount != 2 {
		t.Errorf("RetryCount should be 2, got %d", ev.RetryCount)
	}
}

func TestEventExceedsMaxRetry(t *testing.T) {
	const maxRetry = 3
	ev := &ModelCallFinishedEvent{RetryCount: 3}
	if ev.RetryCount+1 > maxRetry {
		// Should go to DLQ
	} else {
		t.Error("expected retry count to exceed max")
	}
}

func TestEventJSONOmitEmpty(t *testing.T) {
	// Verify omitempty works for optional fields.
	ev := &ModelCallFinishedEvent{
		RequestID: "req_test",
		Status:    "success",
	}
	data, _ := ev.Marshal()

	var raw map[string]interface{}
	json.Unmarshal(data, &raw)

	// Omitempty fields should not appear when empty.
	if _, ok := raw["error_code"]; ok {
		t.Error("error_code should be omitted when empty")
	}
	if _, ok := raw["error_message"]; ok {
		t.Error("error_message should be omitted when empty")
	}
	if _, ok := raw["title_source_content"]; ok {
		t.Error("title_source_content should be omitted when empty")
	}
}
