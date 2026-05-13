package router

import (
	"github.com/gin-gonic/gin"

	"github.com/Daven888/flowchat/internal/handler"
	"github.com/Daven888/flowchat/internal/middleware"
)

// Register sets up all HTTP routes. Dependencies are injected through handlers.
func Register(r *gin.Engine, authH *handler.AuthHandler, userH *handler.UserHandler, sessionH *handler.SessionHandler, messageH *handler.MessageHandler, chatH *handler.ChatHandler, callLogH *handler.CallLogHandler, usageStatH *handler.UsageStatHandler, exportH *handler.ExportHandler, modelH *handler.ModelHandler, credentialH *handler.CredentialHandler, jwtSecret string) {
	r.GET("/ping", handler.Ping)

	api := r.Group("/api/v1")
	{
		// Public model list (no auth required for listing available models)
		api.GET("/models", modelH.List)
		auth := api.Group("/auth")
		{
			auth.POST("/register", authH.Register)
			auth.POST("/login", authH.Login)
		}

		user := api.Group("/user")
		user.Use(middleware.JWTAuth(jwtSecret))
		{
			user.GET("/profile", userH.Profile)
			user.GET("/usage-stats", usageStatH.Get)

			credentials := user.Group("/credentials")
			{
				credentials.GET("", credentialH.List)
				credentials.PUT("/:provider", credentialH.Upsert)
				credentials.DELETE("/:provider", credentialH.Delete)
			}
		}

		chat := api.Group("/chat")
		chat.Use(middleware.JWTAuth(jwtSecret))
		{
			sessions := chat.Group("/sessions")
			{
				sessions.POST("", sessionH.Create)
				sessions.GET("", sessionH.List)
				sessions.GET("/:session_id", sessionH.Get)
				sessions.DELETE("/:session_id", sessionH.Delete)
				sessions.GET("/:session_id/messages", messageH.List)
				sessions.GET("/:session_id/messages/search", messageH.Search)
				sessions.POST("/:session_id/messages/stream", chatH.Stream)
				sessions.GET("/:session_id/export/markdown", exportH.ExportMarkdown)
			}
		}

		callLogs := api.Group("/model-call-logs")
		callLogs.Use(middleware.JWTAuth(jwtSecret))
		{
			callLogs.GET("", callLogH.List)
			callLogs.GET("/:id", callLogH.Get)
		}
	}
}
