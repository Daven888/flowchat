package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Daven888/flowchat/internal/middleware"
	"github.com/Daven888/flowchat/internal/service"
)

// UsageStatHandler handles usage stats query endpoints.
type UsageStatHandler struct {
	usageStatService *service.UsageStatService
}

// NewUsageStatHandler creates a new UsageStatHandler.
func NewUsageStatHandler(usageStatService *service.UsageStatService) *UsageStatHandler {
	return &UsageStatHandler{usageStatService: usageStatService}
}

// Get handles GET /api/v1/user/usage-stats
func (h *UsageStatHandler) Get(c *gin.Context) {
	userID, exists := c.Get(middleware.ContextKeyUserID)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var startDate, endDate time.Time
	if s := c.Query("start_date"); s != "" {
		parsed, err := time.Parse("2006-01-02", s)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start_date format, use YYYY-MM-DD"})
			return
		}
		startDate = parsed
	}
	if s := c.Query("end_date"); s != "" {
		parsed, err := time.Parse("2006-01-02", s)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid end_date format, use YYYY-MM-DD"})
			return
		}
		endDate = parsed
	}

	modelName := c.Query("model_name")

	stats, err := h.usageStatService.List(userID.(int64), startDate, endDate, modelName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	items := make([]UsageStatResponse, 0, len(stats))
	for i := range stats {
		items = append(items, toUsageStatResponse(&stats[i]))
	}

	c.JSON(http.StatusOK, gin.H{"items": items})
}
