package server

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Daven888/flowchat/internal/config"
	"github.com/Daven888/flowchat/internal/handler"
	"github.com/Daven888/flowchat/internal/router"
)

// Run starts the HTTP server with the given configuration and handlers.
func Run(cfg *config.Config, authH *handler.AuthHandler, userH *handler.UserHandler, sessionH *handler.SessionHandler, messageH *handler.MessageHandler, chatH *handler.ChatHandler, callLogH *handler.CallLogHandler, usageStatH *handler.UsageStatHandler, exportH *handler.ExportHandler, modelH *handler.ModelHandler, credentialH *handler.CredentialHandler) error {
	gin.SetMode(gin.ReleaseMode)

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(corsMiddleware())

	router.Register(r, authH, userH, sessionH, messageH, chatH, callLogH, usageStatH, exportH, modelH, credentialH, cfg.JWT.Secret)

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	return r.Run(addr)
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Header("Access-Control-Expose-Headers", "Content-Disposition")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
