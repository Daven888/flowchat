package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/Daven888/flowchat/internal/middleware"
	"github.com/Daven888/flowchat/internal/service"
)

// CallLogHandler handles model call log query endpoints.
type CallLogHandler struct {
	callLogService *service.CallLogService
}

// NewCallLogHandler creates a new CallLogHandler.
func NewCallLogHandler(callLogService *service.CallLogService) *CallLogHandler {
	return &CallLogHandler{callLogService: callLogService}
}

// List handles GET /api/v1/model-call-logs
func (h *CallLogHandler) List(c *gin.Context) {
	userID, exists := c.Get(middleware.ContextKeyUserID)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	status := c.Query("status")
	modelName := c.Query("model_name")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	logs, total, err := h.callLogService.List(userID.(int64), status, modelName, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	items := make([]CallLogResponse, 0, len(logs))
	for i := range logs {
		items = append(items, toCallLogResponse(&logs[i]))
	}

	c.JSON(http.StatusOK, gin.H{
		"items":     items,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// Get handles GET /api/v1/model-call-logs/:id
func (h *CallLogHandler) Get(c *gin.Context) {
	userID, exists := c.Get(middleware.ContextKeyUserID)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	logID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid log id"})
		return
	}

	l, err := h.callLogService.Get(userID.(int64), logID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) || errors.Is(err, service.ErrSessionNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "call log not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"call_log": toCallLogResponse(l)})
}
