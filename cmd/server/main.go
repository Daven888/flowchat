package main

import (
	"context"
	"fmt"
	"os"

	"go.uber.org/zap"

	"github.com/Daven888/flowchat/internal/auth"
	"github.com/Daven888/flowchat/internal/config"
	"github.com/Daven888/flowchat/internal/event"
	"github.com/Daven888/flowchat/internal/handler"
	"github.com/Daven888/flowchat/internal/lock"
	"github.com/Daven888/flowchat/internal/model"
	"github.com/Daven888/flowchat/internal/repository"
	"github.com/Daven888/flowchat/internal/sensitive"
	"github.com/Daven888/flowchat/internal/server"
	"github.com/Daven888/flowchat/internal/service"
	"github.com/Daven888/flowchat/pkg/cryptoutil"
	"github.com/Daven888/flowchat/pkg/logger"
	"github.com/Daven888/flowchat/pkg/mysql"
	appredis "github.com/Daven888/flowchat/pkg/redis"
)

func main() {
	// 1. Load configuration
	cfg, err := config.Load("configs/config.yaml")
	if err != nil {
		fmt.Printf("failed to load config: %v\n", err)
		os.Exit(1)
	}

	// 2. Initialize logger
	if err := logger.Init("info"); err != nil {
		fmt.Printf("failed to init logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	logger.Log.Info("FlowChat server starting...")

	// 3. Initialize credential encryption
	encryptionSecret := os.Getenv(cfg.Credential.EncryptionKeyEnv)
	if encryptionSecret == "" {
		logger.Log.Fatal("credential encryption key not set",
			zap.String("env", cfg.Credential.EncryptionKeyEnv),
		)
	}
	encryptionKey := cryptoutil.DeriveKey(encryptionSecret)
	encryptor, err := cryptoutil.NewAESEncryptor(encryptionKey)
	if err != nil {
		logger.Log.Fatal("failed to init encryptor", zap.Error(err))
	}
	logger.Log.Info("Credential encryption initialized")

	// 4. Initialize MySQL
	if err := mysql.Init(cfg.MySQL); err != nil {
		logger.Log.Fatal("failed to init MySQL", zap.Error(err))
	}
	logger.Log.Info("MySQL connected")

	// 4.5 Run database migrations
	if err := mysql.Migrate(
		&model.User{},
		&model.ChatSession{},
		&model.ChatMessage{},
		&model.ChatSummary{},
		&model.ModelCallLog{},
		&model.UserModelUsageStat{},
		&model.UserProviderCredential{},
	); err != nil {
		logger.Log.Fatal("failed to run migrations", zap.Error(err))
	}
	logger.Log.Info("Database migration completed")

	// 5. Initialize Redis
	if err := appredis.Init(cfg.Redis); err != nil {
		logger.Log.Fatal("failed to init Redis", zap.Error(err))
	}
	defer func() {
		if err := appredis.Close(); err != nil {
			logger.Log.Error("failed to close Redis", zap.Error(err))
		}
	}()
	logger.Log.Info("Redis connected")

	// Build provider names list (for credential validation)
	providerNames := []string{"mock"}
	for name := range cfg.AI.Providers {
		providerNames = append(providerNames, name)
	}

	// 6. Wire dependencies
	jwtCfg := auth.Config{
		Secret:      cfg.JWT.Secret,
		ExpireHours: cfg.JWT.ExpireHours,
	}
	userRepo := repository.NewUserRepository()
	userService := service.NewUserService(userRepo, jwtCfg)
	authHandler := handler.NewAuthHandler(userService)
	userHandler := handler.NewUserHandler(userService)

	providerRegistry := service.NewProviderRegistry(cfg)
	modelService := service.NewModelService(cfg, providerRegistry)

	sessionRepo := repository.NewSessionRepository()
	sessionService := service.NewSessionService(sessionRepo, modelService)
	sessionHandler := handler.NewSessionHandler(sessionService)

	messageRepo := repository.NewMessageRepository()
	messageService := service.NewMessageService(messageRepo, sessionService)
	messageHandler := handler.NewMessageHandler(messageService)

	summaryRepo := repository.NewSummaryRepository()
	compressionService := service.NewCompressionService(messageRepo, summaryRepo, sessionService)

	credentialRepo := repository.NewUserProviderCredentialRepo()
	credentialService := service.NewCredentialService(credentialRepo, encryptor, providerNames)
	credentialHandler := handler.NewCredentialHandler(credentialService)

	lockManager := lock.NewManager(appredis.Client)
	lockTTL := cfg.Chat.SessionLockTTLSeconds
	if lockTTL <= 0 {
		lockTTL = 180
	}
	chatService := service.NewChatService(messageService, sessionService, modelService, credentialService, compressionService, lockManager, lockTTL)

	// Create event publisher for async side tasks.
	eventPublisher := event.NewPublisher()

	callLogRepo := repository.NewCallLogRepository()
	callLogService := service.NewCallLogService(callLogRepo)
	usageStatRepo := repository.NewUsageStatRepository()
	usageStatService := service.NewUsageStatService(usageStatRepo)
	filter := sensitive.New(cfg.SensitiveWords)
	chatHandler := handler.NewChatHandler(chatService, eventPublisher, filter)
	callLogHandler := handler.NewCallLogHandler(callLogService)
	usageStatHandler := handler.NewUsageStatHandler(usageStatService)
	exportService := service.NewExportService(sessionService, messageService)
	exportHandler := handler.NewExportHandler(exportService)
	modelHandler := handler.NewModelHandler(modelService)

	// Start Redis Stream consumer for async side tasks (call logs, usage stats, auto title).
	eventHandler := event.NewModelCallEventHandler(callLogRepo, usageStatRepo, sessionRepo)
	consumer := event.NewConsumer(eventHandler.Handle)
	if err := consumer.Start(context.Background()); err != nil {
		logger.Log.Error("failed to start event consumer", zap.Error(err))
		// Non-fatal: the server still serves requests; side tasks will be processed
		// when the consumer is restarted or when pending messages are claimed later.
	}

	// 7. Start HTTP server
	logger.Log.Info("HTTP server starting", zap.Int("port", cfg.Server.Port))
	if err := server.Run(cfg, authHandler, userHandler, sessionHandler, messageHandler, chatHandler, callLogHandler, usageStatHandler, exportHandler, modelHandler, credentialHandler); err != nil {
		logger.Log.Fatal("failed to run server", zap.Error(err))
	}
}
