package service

import (
	"context"
	"fmt"
	"strings"

	"go.uber.org/zap"

	"github.com/Daven888/flowchat/internal/config"
	"github.com/Daven888/flowchat/internal/model"
	"github.com/Daven888/flowchat/internal/provider"
	"github.com/Daven888/flowchat/internal/repository"
	"github.com/Daven888/flowchat/pkg/logger"
)

// CompressionService compresses long conversation history into a summary,
// enabling models to retain context across many turns without exceeding token limits.
type CompressionService struct {
	msgRepo     *repository.MessageRepository
	summaryRepo *repository.SummaryRepository
	sessionSvc  *SessionService
}

// NewCompressionService creates a new CompressionService.
func NewCompressionService(msgRepo *repository.MessageRepository, summaryRepo *repository.SummaryRepository, sessionSvc *SessionService) *CompressionService {
	return &CompressionService{
		msgRepo:     msgRepo,
		summaryRepo: summaryRepo,
		sessionSvc:  sessionSvc,
	}
}

// GetContextWithSummary returns the model context for a session.
// If the session has enough completed messages to trigger compression, it generates
// a summary of older messages and returns [system summary] + [recent N messages].
// Otherwise it returns only the recent N messages.
// On summary generation failure, it falls back to recent messages only.
func (s *CompressionService) GetContextWithSummary(ctx context.Context, sessionID, userID int64, modelCfg config.ModelConfig, p provider.ChatProvider, apiKey string) ([]provider.ProviderMessage, error) {
	// Validate session ownership.
	if _, err := s.sessionSvc.Get(userID, sessionID); err != nil {
		return nil, err
	}

	keepRecent := modelCfg.MaxContextMessages
	if keepRecent <= 0 {
		keepRecent = 10
	}

	// Load recent completed messages (ASC order).
	recent, err := s.msgRepo.FindCompletedBySessionID(sessionID, keepRecent)
	if err != nil {
		return nil, err
	}

	// If compression is disabled, return recent messages as-is.
	cc := modelCfg.ContextCompression
	if !cc.Enabled || cc.MaxMessagesBeforeCompress <= 0 {
		return toProviderMessages(recent), nil
	}

	// Count total completed messages.
	total, err := s.msgRepo.CountCompletedBySessionID(sessionID)
	if err != nil {
		return nil, err
	}

	// Not enough messages to trigger compression.
	if total <= int64(cc.MaxMessagesBeforeCompress) {
		return toProviderMessages(recent), nil
	}

	// Load all completed messages to find the ones to summarize.
	allCompleted, err := s.msgRepo.FindAllCompletedBySessionID(sessionID)
	if err != nil {
		return nil, err
	}

	// The messages to summarize are all completed messages excluding the most recent N.
	if len(allCompleted) <= keepRecent {
		return toProviderMessages(recent), nil
	}
	toSummarize := allCompleted[:len(allCompleted)-keepRecent]

	// If nothing to summarize, return recent only.
	if len(toSummarize) == 0 {
		return toProviderMessages(recent), nil
	}

	// Generate summary via the provider.
	summaryText, err := s.generateSummary(ctx, p, modelCfg.APIModel, apiKey, toSummarize)
	if err != nil {
		if logger.Log != nil {
			logger.Log.Warn("summary generation failed, falling back to recent messages",
				zap.Error(err),
				zap.Int64("session_id", sessionID),
			)
		}
		return toProviderMessages(recent), nil
	}

	// Persist the summary.
	lastID := toSummarize[len(toSummarize)-1].ID
	if err := s.summaryRepo.Upsert(&model.ChatSummary{
		SessionID:     sessionID,
		Content:       summaryText,
		LastMessageID: lastID,
	}); err != nil {
		if logger.Log != nil {
			logger.Log.Warn("failed to persist summary, continuing with in-memory summary",
				zap.Error(err),
				zap.Int64("session_id", sessionID),
			)
		}
		// Still return the summary in context even if persistence fails.
	}

	// Build context: system summary + recent messages.
	result := make([]provider.ProviderMessage, 0, 1+len(recent))
	result = append(result, provider.ProviderMessage{
		Role:    model.MessageRoleSystem,
		Content: summaryText,
	})
	for _, m := range recent {
		result = append(result, provider.ProviderMessage{
			Role:    m.Role,
			Content: m.Content,
		})
	}
	return result, nil
}

// generateSummary calls the provider to produce a summary of the given messages.
func (s *CompressionService) generateSummary(ctx context.Context, p provider.ChatProvider, apiModel, apiKey string, messages []model.ChatMessage) (string, error) {
	prompt := buildSummaryPrompt()
	conversationText := formatMessagesForSummary(messages)

	summaryReq := provider.ChatRequest{
		ModelName: apiModel,
		Messages: []provider.ProviderMessage{
			{Role: model.MessageRoleSystem, Content: prompt},
			{Role: model.MessageRoleUser, Content: conversationText},
		},
		APIKey: apiKey,
	}

	ch, err := p.StreamChat(ctx, summaryReq)
	if err != nil {
		return "", fmt.Errorf("summary provider call failed: %w", err)
	}

	var fullContent strings.Builder
	for chunk := range ch {
		if chunk.Err != nil {
			return "", fmt.Errorf("summary stream error: %w", chunk.Err)
		}
		if chunk.Done {
			break
		}
		fullContent.WriteString(chunk.Content)
	}

	result := strings.TrimSpace(fullContent.String())
	if result == "" {
		return "", fmt.Errorf("summary provider returned empty content")
	}
	return result, nil
}

// buildSummaryPrompt returns the system prompt for summary generation.
func buildSummaryPrompt() string {
	return `你是一个对话摘要助手。请根据以下对话历史，生成简洁的摘要。

要求：
1. 总结用户的主要需求和目标
2. 记录已经确定的事实和决策
3. 列出重要的约束条件
4. 列出已完成的事项
5. 列出待解决的问题

注意：
- 不要输出寒暄或问候语
- 不要编造对话中未出现的信息
- 使用中文输出
- 控制在简洁的篇幅内`
}

// formatMessagesForSummary formats a slice of completed messages for the summary prompt.
func formatMessagesForSummary(messages []model.ChatMessage) string {
	var b strings.Builder
	for _, m := range messages {
		switch m.Role {
		case model.MessageRoleUser:
			b.WriteString("用户：")
		case model.MessageRoleAssistant:
			b.WriteString("助手：")
		default:
			continue
		}
		b.WriteString(m.Content)
		b.WriteString("\n\n")
	}
	return b.String()
}

// toProviderMessages converts a slice of ChatMessage to ProviderMessage.
func toProviderMessages(msgs []model.ChatMessage) []provider.ProviderMessage {
	result := make([]provider.ProviderMessage, 0, len(msgs))
	for _, m := range msgs {
		result = append(result, provider.ProviderMessage{
			Role:    m.Role,
			Content: m.Content,
		})
	}
	return result
}
