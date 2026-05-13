package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

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

	// Parse before_id
	var beforeID int64
	if v := c.Query("before_id"); v != "" {
		beforeID, err = strconv.ParseInt(v, 10, 64)
		if err != nil || beforeID < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid before_id"})
			return
		}
	}

	// Parse limit: default 50, max 100
	limit := 50
	if v := c.Query("limit"); v != "" {
		limit, err = strconv.Atoi(v)
		if err != nil || limit <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid limit"})
			return
		}
		if limit > 100 {
			limit = 100
		}
	}

	messages, hasMore, nextBeforeID, err := h.messageService.ListMessages(sessionID, userID.(int64), beforeID, limit)
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

	c.JSON(http.StatusOK, gin.H{
		"messages":       items,
		"has_more":       hasMore,
		"next_before_id": nextBeforeID,
	})
}

// Search handles GET /api/v1/chat/sessions/:session_id/messages/search
func (h *MessageHandler) Search(c *gin.Context) {
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

	query := strings.TrimSpace(c.Query("q"))
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "search query cannot be empty"})
		return
	}

	limit := 50
	if v := c.Query("limit"); v != "" {
		limit, err = strconv.Atoi(v)
		if err != nil || limit <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid limit"})
			return
		}
		if limit > 100 {
			limit = 100
		}
	}

	messages, err := h.messageService.SearchMessages(sessionID, userID.(int64), query, limit)
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
