package provider

import "context"

// ChatProvider abstracts a model service for streaming chat.
type ChatProvider interface {
	StreamChat(ctx context.Context, req ChatRequest) (<-chan ChatChunk, error)
}

// ChatRequest contains all data needed for a single model call.
type ChatRequest struct {
	RequestID string
	ModelName string
	Messages  []ProviderMessage
}

// ProviderMessage is a single message in the provider's format.
type ProviderMessage struct {
	Role    string
	Content string
}

// ChatChunk represents a piece of streaming output from a provider.
type ChatChunk struct {
	Content          string
	Done             bool
	Err              error
	FinishReason     string
	PromptTokens     int
	CompletionTokens int
}
