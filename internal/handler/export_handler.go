package handler

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/Daven888/flowchat/internal/middleware"
	"github.com/Daven888/flowchat/internal/service"
)

// ExportHandler handles Markdown export endpoints.
type ExportHandler struct {
	exportService *service.ExportService
}

// NewExportHandler creates a new ExportHandler.
func NewExportHandler(exportService *service.ExportService) *ExportHandler {
	return &ExportHandler{exportService: exportService}
}

// ExportMarkdown handles GET /api/v1/chat/sessions/:session_id/export/markdown
func (h *ExportHandler) ExportMarkdown(c *gin.Context) {
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

	markdown, err := h.exportService.ExportMarkdown(userID.(int64), sessionID)
	if err != nil {
		if errors.Is(err, service.ErrSessionNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	filename := fmt.Sprintf("flowchat-session-%d.md", sessionID)
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Data(http.StatusOK, "text/markdown; charset=utf-8", []byte(markdown))
}
