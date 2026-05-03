package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Daven888/flowchat/internal/middleware"
	"github.com/Daven888/flowchat/internal/model"
	"github.com/Daven888/flowchat/internal/sensitive"
	"github.com/Daven888/flowchat/internal/service"
)

// ChatHandler handles the streaming chat endpoint.
type ChatHandler struct {
	chatService      *service.ChatService
	callLogService   *service.CallLogService
	usageStatService *service.UsageStatService
	filter           *sensitive.Filter
}

// NewChatHandler creates a new ChatHandler.
func NewChatHandler(chatService *service.ChatService, callLogService *service.CallLogService, usageStatService *service.UsageStatService, filter *sensitive.Filter) *ChatHandler {
	return &ChatHandler{chatService: chatService, callLogService: callLogService, usageStatService: usageStatService, filter: filter}
}

// StreamRequest is the request body for the streaming chat endpoint.
type StreamRequest struct {
	Content string `json:"content" binding:"required"`
}

// Stream handles POST /api/v1/chat/sessions/:session_id/messages/stream
func (h *ChatHandler) Stream(c *gin.Context) {
	userID, exists := c.Get(middleware.ContextKeyUserID)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	sessionID, err := strconv.ParseInt(c.Param("session_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session id"})
		return
	}

	var req StreamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Trim and validate content before SSE headers
	content := strings.TrimSpace(req.Content)
	if content == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "content cannot be empty"})
		return
	}

	// Sensitive word check
	if h.filter != nil && h.filter.Contains(content) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error_code": "SENSITIVE_CONTENT",
			"message":    "输入内容包含不支持的内容",
		})
		return
	}

	// From here on, switch to SSE mode
	c.Writer.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Writer.WriteHeader(http.StatusOK)

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		return
	}

	ctx := c.Request.Context()
	uid := userID.(int64)

	handle, err := h.chatService.BeginStream(ctx, uid, sessionID, req.Content)
	if err != nil {
		msg := err.Error()
		if errors.Is(err, service.ErrSessionGenerating) {
			msg = "当前会话正在生成回复，请稍后再试"
		}
		writeSSEEvent(c.Writer, "error", map[string]string{"error": msg})
		flusher.Flush()
		return
	}
	// Release the session lock on all exit paths
	defer func() {
		_ = h.chatService.ReleaseLock(sessionID, handle.RequestID)
		handle.Cleanup()
	}()

	// Track streaming outcome for call log (written in defer)
	var (
		streamStatus   = model.CallLogStatusFailed
		streamTokens   int
		streamErrorMsg string
		streamFinish   string
		promptTokens   int
	)
	defer func() {
		finishedAt := time.Now()
		latencyMs := finishedAt.Sub(handle.StartedAt).Milliseconds()
		h.callLogService.Log(service.CreateCallLogParams{
			RequestID:        handle.RequestID,
			UserID:           uid,
			SessionID:        sessionID,
			Provider:         handle.ProviderName,
			ModelName:        handle.ModelName,
			Status:           streamStatus,
			PromptTokens:     promptTokens,
			CompletionTokens: streamTokens,
			LatencyMs:        latencyMs,
			ErrorCode:        streamStatus,
			ErrorMessage:     streamErrorMsg,
			FinishReason:     streamFinish,
			StartedAt:        handle.StartedAt,
			FinishedAt:       finishedAt,
		})

		h.usageStatService.Update(service.UpdateUsageParams{
			UserID:           uid,
			ModelName:        handle.ModelName,
			Status:           streamStatus,
			PromptTokens:     promptTokens,
			CompletionTokens: streamTokens,
		})

		// Auto-generate session title on first successful completion
		if streamStatus == model.CallLogStatusSuccess {
			h.chatService.AutoGenerateTitle(sessionID, uid, content)
		}
	}()

	// Send meta event
	writeSSEEvent(c.Writer, "meta", map[string]interface{}{
		"request_id":           handle.RequestID,
		"assistant_message_id": handle.AssistantMessageID,
	})
	flusher.Flush()

	// Stream chunks
	var fullContent strings.Builder
	var doneReached bool

	for chunk := range handle.Chunks {
		if chunk.Err != nil {
			streamErrorMsg = chunk.Err.Error()
			_ = h.chatService.Fail(sessionID, uid, handle.AssistantMessageID, streamErrorMsg)
			writeSSEEvent(c.Writer, "error", map[string]string{"error": streamErrorMsg})
			flusher.Flush()
			return
		}

		if chunk.Done {
			doneReached = true
			promptTokens = chunk.PromptTokens
			streamTokens = chunk.CompletionTokens
			streamFinish = chunk.FinishReason
			break
		}

		fullContent.WriteString(chunk.Content)
		writeSSEEvent(c.Writer, "message", map[string]string{"content": chunk.Content})
		flusher.Flush()
	}

	// Handle completion or cancellation
	if doneReached {
		streamStatus = model.CallLogStatusSuccess
		_ = h.chatService.Complete(sessionID, uid, handle.AssistantMessageID, fullContent.String(), streamTokens)
		writeSSEEvent(c.Writer, "done", map[string]string{"message": "completed"})
		flusher.Flush()
		return
	}

	// Channel closed without Done: check if context was cancelled
	select {
	case <-ctx.Done():
		streamStatus = model.CallLogStatusCancelled
		_ = h.chatService.Cancel(sessionID, uid, handle.AssistantMessageID)
	default:
		streamErrorMsg = "stream closed unexpectedly"
		_ = h.chatService.Fail(sessionID, uid, handle.AssistantMessageID, streamErrorMsg)
		writeSSEEvent(c.Writer, "error", map[string]string{"error": streamErrorMsg})
		flusher.Flush()
	}
}

// writeSSEEvent writes a single SSE event to the response writer.
func writeSSEEvent(w http.ResponseWriter, event string, data interface{}) {
	jsonData, _ := json.Marshal(data)
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, string(jsonData))
}
