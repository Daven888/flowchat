package openai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/Daven888/flowchat/internal/provider"
)

// Provider implements ChatProvider for OpenAI-compatible APIs (e.g., DeepSeek).
type Provider struct {
	baseURL string
	client  *http.Client
}

// New creates a new OpenAI-compatible Provider.
func New(baseURL string) *Provider {
	return &Provider{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{},
	}
}

// StreamChat sends a streaming chat completion request and returns a channel of chunks.
func (p *Provider) StreamChat(ctx context.Context, req provider.ChatRequest) (<-chan provider.ChatChunk, error) {
	if req.APIKey == "" {
		return nil, fmt.Errorf("openai provider: api key not configured")
	}

	msgs := make([]map[string]string, len(req.Messages))
	for i, m := range req.Messages {
		msgs[i] = map[string]string{"role": m.Role, "content": m.Content}
	}

	body := map[string]interface{}{
		"model":    req.ModelName,
		"messages": msgs,
		"stream":   true,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat/completions", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+req.APIKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("openai api error: status=%d body=%s", resp.StatusCode, string(respBody))
	}

	ch := make(chan provider.ChatChunk, 10)

	go func() {
		defer close(ch)
		defer resp.Body.Close()

		scanner := bufio.NewScanner(resp.Body)
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

		var fullContent strings.Builder

		for scanner.Scan() {
			// Check context cancellation
			select {
			case <-ctx.Done():
				return
			default:
			}

			line := scanner.Text()
			if !strings.HasPrefix(line, "data: ") {
				continue
			}

			data := strings.TrimPrefix(line, "data: ")

			if data == "[DONE]" {
				select {
				case ch <- provider.ChatChunk{
					Done:             true,
					FinishReason:     "stop",
					PromptTokens:     estimateTokens(req),
					CompletionTokens: estimateTextTokens(fullContent.String()),
				}:
				case <-ctx.Done():
				}
				return
			}

			var chunk openAIStreamChunk
			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				continue
			}

			if len(chunk.Choices) > 0 {
				delta := chunk.Choices[0].Delta.Content
				if delta != "" {
					fullContent.WriteString(delta)
					select {
					case ch <- provider.ChatChunk{Content: delta}:
					case <-ctx.Done():
						return
					}
				}
			}
		}

		if err := scanner.Err(); err != nil {
			select {
			case ch <- provider.ChatChunk{Err: err}:
			case <-ctx.Done():
			}
			return
		}

		// Stream ended without [DONE] marker — treat as successful completion
		select {
		case ch <- provider.ChatChunk{
			Done:             true,
			FinishReason:     "stop",
			PromptTokens:     estimateTokens(req),
			CompletionTokens: estimateTextTokens(fullContent.String()),
		}:
		case <-ctx.Done():
		}
	}()

	return ch, nil
}

// openAIStreamChunk represents a single SSE data chunk from OpenAI-compatible APIs.
type openAIStreamChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
}

// estimateTokens estimates token count from the chat request messages.
func estimateTokens(req provider.ChatRequest) int {
	total := 0
	for _, m := range req.Messages {
		total += utf8.RuneCountInString(m.Content)
	}
	return total
}

// estimateTextTokens estimates token count from a text string.
func estimateTextTokens(s string) int {
	return utf8.RuneCountInString(s)
}
