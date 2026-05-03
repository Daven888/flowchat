package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Ping responds with a simple pong message.
func Ping(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "pong"})
}
