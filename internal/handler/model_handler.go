package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Daven888/flowchat/internal/service"
)

// ModelHandler handles model whitelist endpoints.
type ModelHandler struct {
	modelService *service.ModelService
}

// NewModelHandler creates a new ModelHandler.
func NewModelHandler(modelService *service.ModelService) *ModelHandler {
	return &ModelHandler{modelService: modelService}
}

// List handles GET /api/v1/models
func (h *ModelHandler) List(c *gin.Context) {
	models := h.modelService.GetEnabledModels()

	type modelItem struct {
		Name     string `json:"name"`
		Provider string `json:"provider"`
		Enabled  bool   `json:"enabled"`
	}
	items := make([]modelItem, 0, len(models))
	for _, m := range models {
		items = append(items, modelItem{
			Name:     m.Name,
			Provider: m.Provider,
			Enabled:  m.Enabled,
		})
	}

	c.JSON(http.StatusOK, gin.H{"models": items})
}
