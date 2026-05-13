package service

import (
	"errors"

	"github.com/Daven888/flowchat/internal/config"
	"github.com/Daven888/flowchat/internal/provider"
	mp "github.com/Daven888/flowchat/internal/provider/mock"
	op "github.com/Daven888/flowchat/internal/provider/openai"
)

var (
	ErrModelNotFound    = errors.New("model not found")
	ErrModelDisabled    = errors.New("model is disabled")
	ErrProviderNotFound = errors.New("provider not found")
)

// ProviderRegistry creates and manages ChatProvider instances.
type ProviderRegistry struct {
	providers map[string]provider.ChatProvider
}

// NewProviderRegistry builds a ProviderRegistry from config.
func NewProviderRegistry(cfg *config.Config) *ProviderRegistry {
	r := &ProviderRegistry{providers: make(map[string]provider.ChatProvider)}

	r.providers["mock"] = mp.New()

	for name, pc := range cfg.AI.Providers {
		switch pc.Type {
		case "openai_compatible":
			r.providers[name] = op.New(pc.BaseURL)
		}
	}

	return r
}

// Get returns the ChatProvider for the given provider name.
func (r *ProviderRegistry) Get(name string) (provider.ChatProvider, error) {
	p, ok := r.providers[name]
	if !ok {
		return nil, ErrProviderNotFound
	}
	return p, nil
}

// ModelService manages model configuration and lookup.
type ModelService struct {
	models   []config.ModelConfig
	registry *ProviderRegistry
}

// NewModelService creates a new ModelService.
func NewModelService(cfg *config.Config, registry *ProviderRegistry) *ModelService {
	return &ModelService{models: cfg.Models, registry: registry}
}

// GetEnabledModels returns all enabled model configs.
func (s *ModelService) GetEnabledModels() []config.ModelConfig {
	var enabled []config.ModelConfig
	for _, m := range s.models {
		if m.Enabled {
			enabled = append(enabled, m)
		}
	}
	return enabled
}

// GetModelConfig looks up a model config by name and validates it is enabled.
func (s *ModelService) GetModelConfig(modelName string) (config.ModelConfig, error) {
	for _, m := range s.models {
		if m.Name == modelName {
			if !m.Enabled {
				return config.ModelConfig{}, ErrModelDisabled
			}
			return m, nil
		}
	}
	return config.ModelConfig{}, ErrModelNotFound
}

// GetProvider returns the ChatProvider for a given model config.
func (s *ModelService) GetProvider(modelCfg config.ModelConfig) (provider.ChatProvider, error) {
	return s.registry.Get(modelCfg.Provider)
}
