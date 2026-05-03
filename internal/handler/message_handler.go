package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/Daven888/flowchat/internal/middleware"
	"github.com/Daven888/flowchat/internal/service"
)

// MessageHandler handles message query endpoints.
type MessageHandler struct {
	messageService *service.MessageService
}

// NewMessageHandler creates a new MessageHandler.
func NewMessageHandler(messageService *service.MessageService) *MessageHandler {
	return &MessageHandler{messageService: messageService}
}

// List handles GET /api/v1/chat/sessions/:session_id/messages
func (h *MessageHandler) List(c *gin.Context) {
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

	messages, err := h.messageService.ListMessages(sessionID, userID.(int64))
	if err != nil {
		if errors.Is(err, service.ErrSessionNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	items := make([]MessageResponse, 0, len(messages))
	for i := range messages {
		items = append(items, toMessageResponse(&messages[i]))
	}

	c.JSON(http.StatusOK, gin.H{"messages": items})
}
