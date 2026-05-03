package config

import (
	"github.com/spf13/viper"

	"github.com/Daven888/flowchat/pkg/mysql"
	"github.com/Daven888/flowchat/pkg/redis"
)

// Config holds all configuration for the application.
type Config struct {
	Server         ServerConfig              `mapstructure:"server"`
	MySQL          mysql.Config              `mapstructure:"mysql"`
	Redis          redis.Config              `mapstructure:"redis"`
	JWT            JWTConfig                 `mapstructure:"jwt"`
	Chat           ChatConfig                `mapstructure:"chat"`
	AI             AIConfig                  `mapstructure:"ai"`
	Models         []ModelConfig             `mapstructure:"models"`
	SensitiveWords []string                  `mapstructure:"sensitive_words"`
}

// ServerConfig holds HTTP server configuration.
type ServerConfig struct {
	Port int `mapstructure:"port"`
}

// JWTConfig holds JWT signing configuration.
type JWTConfig struct {
	Secret      string `mapstructure:"secret"`
	ExpireHours int    `mapstructure:"expire_hours"`
}

// ChatConfig holds chat-related configuration.
type ChatConfig struct {
	SessionLockTTLSeconds int `mapstructure:"session_lock_ttl_seconds"`
	MaxMessageLength      int `mapstructure:"max_message_length"`
}

// AIConfig holds AI provider configurations.
type AIConfig struct {
	Providers map[string]ProviderConfig `mapstructure:"providers"`
}

// ProviderConfig holds a single AI provider's connection settings.
type ProviderConfig struct {
	Type      string `mapstructure:"type"`
	BaseURL   string `mapstructure:"base_url"`
	APIKeyEnv string `mapstructure:"api_key_env"`
}

// ModelConfig holds a single model's runtime configuration.
type ModelConfig struct {
	Name               string `mapstructure:"name"`
	Provider           string `mapstructure:"provider"`
	APIModel           string `mapstructure:"api_model"`
	Enabled            bool   `mapstructure:"enabled"`
	MaxContextMessages int    `mapstructure:"max_context_messages"`
	TimeoutSeconds     int    `mapstructure:"timeout_seconds"`
	MaxRetries         int    `mapstructure:"max_retries"`
}

// Load reads the YAML config file and unmarshals it into a Config struct.
func Load(path string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("yaml")

	// Enable environment variable substitution (e.g. ${OPENAI_API_KEY})
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
