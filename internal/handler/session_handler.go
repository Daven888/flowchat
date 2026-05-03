package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/Daven888/flowchat/internal/middleware"
	"github.com/Daven888/flowchat/internal/service"
)

// SessionHandler handles chat session endpoints.
type SessionHandler struct {
	sessionService *service.SessionService
}

// NewSessionHandler creates a new SessionHandler.
func NewSessionHandler(sessionService *service.SessionService) *SessionHandler {
	return &SessionHandler{sessionService: sessionService}
}

// CreateSessionRequest defines the create session request body.
type CreateSessionRequest struct {
	Title     string `json:"title"`
	ModelName string `json:"model_name" binding:"required"`
}

// Create handles POST /api/v1/chat/sessions
func (h *SessionHandler) Create(c *gin.Context) {
	userID, exists := c.Get(middleware.ContextKeyUserID)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req CreateSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	session, err := h.sessionService.Create(userID.(int64), req.Title, req.ModelName)
	if err != nil {
		if errors.Is(err, service.ErrInvalidModelName) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid model name"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"session": toSessionResponse(session)})
}

// List handles GET /api/v1/chat/sessions
func (h *SessionHandler) List(c *gin.Context) {
	userID, exists := c.Get(middleware.ContextKeyUserID)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	sessions, err := h.sessionService.List(userID.(int64))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	items := make([]SessionResponse, 0, len(sessions))
	for i := range sessions {
		items = append(items, toSessionResponse(&sessions[i]))
	}

	c.JSON(http.StatusOK, gin.H{"sessions": items})
}

// Get handles GET /api/v1/chat/sessions/:session_id
func (h *SessionHandler) Get(c *gin.Context) {
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

	session, err := h.sessionService.Get(userID.(int64), sessionID)
	if err != nil {
		if errors.Is(err, service.ErrSessionNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"session": toSessionResponse(session)})
}

// Delete handles DELETE /api/v1/chat/sessions/:session_id
func (h *SessionHandler) Delete(c *gin.Context) {
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

	if err := h.sessionService.Delete(userID.(int64), sessionID); err != nil {
		if errors.Is(err, service.ErrSessionNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "session deleted"})
}
