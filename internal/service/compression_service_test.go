package service

import (
	"strings"
	"testing"

	"github.com/Daven888/flowchat/internal/model"
	"github.com/Daven888/flowchat/internal/provider"
)

func TestBuildSummaryPrompt(t *testing.T) {
	prompt := buildSummaryPrompt()
	if prompt == "" {
		t.Fatal("summary prompt is empty")
	}
	requiredKeys := []string{
		"主要需求",
		"事实",
		"约束",
		"已完成",
		"待解决",
	}
	for _, key := range requiredKeys {
		if !strings.Contains(prompt, key) {
			t.Errorf("prompt should contain %q", key)
		}
	}
	if strings.Contains(prompt, "你好") || strings.Contains(prompt, "Hello") {
		t.Error("prompt should not contain greetings")
	}
}

func TestFormatMessagesForSummary(t *testing.T) {
	msgs := []model.ChatMessage{
		{ID: 1, Role: model.MessageRoleUser, Content: "你好"},
		{ID: 2, Role: model.MessageRoleAssistant, Content: "你好，有什么可以帮助你的？"},
		{ID: 3, Role: model.MessageRoleUser, Content: "帮我写代码"},
	}
	result := formatMessagesForSummary(msgs)

	if !strings.Contains(result, "用户：你好") {
		t.Error("should contain user message")
	}
	if !strings.Contains(result, "助手：你好，有什么可以帮助你的？") {
		t.Error("should contain assistant message")
	}
	if !strings.Contains(result, "用户：帮我写代码") {
		t.Error("should contain second user message")
	}
}

func TestFormatMessagesForSummaryEmpty(t *testing.T) {
	result := formatMessagesForSummary(nil)
	if result != "" {
		t.Errorf("expected empty string for nil input, got %q", result)
	}
	result = formatMessagesForSummary([]model.ChatMessage{})
	if result != "" {
		t.Errorf("expected empty string for empty input, got %q", result)
	}
}

func TestFormatMessagesForSummarySkipsSystem(t *testing.T) {
	msgs := []model.ChatMessage{
		{ID: 1, Role: model.MessageRoleSystem, Content: "should be skipped"},
		{ID: 2, Role: model.MessageRoleUser, Content: "visible"},
	}
	result := formatMessagesForSummary(msgs)
	if strings.Contains(result, "should be skipped") {
		t.Error("system messages should be skipped in summary formatting")
	}
	if !strings.Contains(result, "visible") {
		t.Error("user messages should appear")
	}
}

func TestToProviderMessages(t *testing.T) {
	input := []model.ChatMessage{
		{ID: 1, Role: model.MessageRoleUser, Content: "hello"},
		{ID: 2, Role: model.MessageRoleAssistant, Content: "hi"},
		{ID: 3, Role: model.MessageRoleUser, Content: "how are you"},
	}
	result := toProviderMessages(input)

	if len(result) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(result))
	}
	for i, m := range result {
		if m.Role != input[i].Role {
			t.Errorf("index %d: Role = %q, want %q", i, m.Role, input[i].Role)
		}
		if m.Content != input[i].Content {
			t.Errorf("index %d: Content = %q, want %q", i, m.Content, input[i].Content)
		}
	}
}

func TestToProviderMessagesEmpty(t *testing.T) {
	result := toProviderMessages(nil)
	if len(result) != 0 {
		t.Errorf("expected empty slice for nil, got %d elements", len(result))
	}

	result = toProviderMessages([]model.ChatMessage{})
	if len(result) != 0 {
		t.Errorf("expected empty slice for empty input, got %d elements", len(result))
	}
}

func TestToProviderMessagesPreservesOrder(t *testing.T) {
	input := []model.ChatMessage{
		{ID: 10, Role: model.MessageRoleUser, Content: "first"},
		{ID: 20, Role: model.MessageRoleAssistant, Content: "second"},
		{ID: 30, Role: model.MessageRoleUser, Content: "third"},
	}
	result := toProviderMessages(input)

	// Order must be ASC (same as input).
	expectedOrder := []int64{10, 20, 30}
	for i, m := range result {
		// Verify indirectly via content
		if m.Content != input[i].Content {
			t.Errorf("index %d: order mismatch, content %q", i, m.Content)
		}
		_ = expectedOrder
	}
}

func TestProviderMessageStructure(t *testing.T) {
	// Verify ProviderMessage has the expected fields used by GetContextWithSummary.
	msg := provider.ProviderMessage{
		Role:    model.MessageRoleSystem,
		Content: "test summary",
	}
	if msg.Role != "system" {
		t.Errorf("Role = %q, want system", msg.Role)
	}
	_ = msg
}
