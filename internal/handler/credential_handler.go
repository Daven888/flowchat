package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Daven888/flowchat/internal/middleware"
	"github.com/Daven888/flowchat/internal/service"
)

// CredentialHandler handles user API key credential endpoints.
type CredentialHandler struct {
	credentialService *service.CredentialService
}

// NewCredentialHandler creates a new CredentialHandler.
func NewCredentialHandler(credentialService *service.CredentialService) *CredentialHandler {
	return &CredentialHandler{credentialService: credentialService}
}

// UpsertCredentialRequest is the request body for creating or updating a credential.
type UpsertCredentialRequest struct {
	APIKey string `json:"api_key" binding:"required"`
}

// List handles GET /api/v1/user/credentials
func (h *CredentialHandler) List(c *gin.Context) {
	userID, exists := c.Get(middleware.ContextKeyUserID)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	statuses, err := h.credentialService.ListStatus(userID.(int64))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"credentials": statuses})
}

// Upsert handles PUT /api/v1/user/credentials/:provider
func (h *CredentialHandler) Upsert(c *gin.Context) {
	userID, exists := c.Get(middleware.ContextKeyUserID)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	providerName := c.Param("provider")

	var req UpsertCredentialRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	keySuffix, err := h.credentialService.Upsert(userID.(int64), providerName, req.APIKey)
	if err != nil {
		if errors.Is(err, service.ErrMockNoKey) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "mock provider does not require an api key"})
			return
		}
		if errors.Is(err, service.ErrInvalidProvider) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid provider: " + providerName})
			return
		}
		if errors.Is(err, service.ErrAPIKeyEmpty) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "api_key cannot be empty"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "credential saved",
		"provider_name": providerName,
		"key_suffix":    keySuffix,
	})
}

// Delete handles DELETE /api/v1/user/credentials/:provider
func (h *CredentialHandler) Delete(c *gin.Context) {
	userID, exists := c.Get(middleware.ContextKeyUserID)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	providerName := c.Param("provider")

	if err := h.credentialService.Delete(userID.(int64), providerName); err != nil {
		if errors.Is(err, service.ErrMockNoKey) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "mock provider does not require an api key"})
			return
		}
		if errors.Is(err, service.ErrInvalidProvider) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid provider: " + providerName})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "credential deleted",
		"provider_name": providerName,
	})
}
