package mock

import (
	"context"
	"fmt"
	"time"
	"unicode/utf8"

	"github.com/Daven888/flowchat/internal/provider"
)

// Provider is a mock ChatProvider that simulates streaming output locally.
type Provider struct{}

// New creates a new mock Provider.
func New() *Provider {
	return &Provider{}
}

// StreamChat generates a simulated reply based on the last user message,
// streaming small chunks every 100ms. It respects context cancellation.
// When the first message is a system message, it treats the request as a
// summarization call and returns a short fixed summary.
func (p *Provider) StreamChat(ctx context.Context, req provider.ChatRequest) (<-chan provider.ChatChunk, error) {
	// Summarization mode: first message is a system prompt.
	if len(req.Messages) > 0 && req.Messages[0].Role == "system" {
		reply := "Summary: The user and assistant discussed various topics. Key points were identified. Pending items may remain."
		completionTokens := estimateTokens(reply)

		ch := make(chan provider.ChatChunk, 1)
		go func() {
			defer close(ch)
			ch <- provider.ChatChunk{
				Done:             true,
				FinishReason:     "stop",
				PromptTokens:     0,
				CompletionTokens: completionTokens,
			}
		}()
		return ch, nil
	}

	// Find the last user message
	userMsg := ""
	for i := len(req.Messages) - 1; i >= 0; i-- {
		if req.Messages[i].Role == "user" {
			userMsg = req.Messages[i].Content
			break
		}
	}

	reply := fmt.Sprintf("这是 Mock Provider 生成的回复。你刚才的问题是：%s", userMsg)

	promptTokens := estimateTokens(userMsg)
	completionTokens := estimateTokens(reply)

	ch := make(chan provider.ChatChunk, 10)

	go func() {
		defer close(ch)

		runes := []rune(reply)
		const chunkSize = 5

		for i := 0; i < len(runes); i += chunkSize {
			select {
			case <-ctx.Done():
				return
			default:
			}

			time.Sleep(100 * time.Millisecond)

			end := i + chunkSize
			if end > len(runes) {
				end = len(runes)
			}

			chunk := provider.ChatChunk{
				Content: string(runes[i:end]),
			}

			select {
			case ch <- chunk:
			case <-ctx.Done():
				return
			}
		}

		// Final chunk: Done=true
		select {
		case ch <- provider.ChatChunk{
			Done:             true,
			FinishReason:     "stop",
			PromptTokens:     promptTokens,
			CompletionTokens: completionTokens,
		}:
		case <-ctx.Done():
		}
	}()

	return ch, nil
}

// estimateTokens provides a rough token count based on rune length.
func estimateTokens(s string) int {
	return utf8.RuneCountInString(s)
}
