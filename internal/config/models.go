// ABOUTME: Model registry resolution: built-in + custom, provider detection
// ABOUTME: Resolves model IDs from config, CLI flags, and built-in definitions

package config

import (
	"fmt"
	"maps"
	"strings"

	"github.com/mauromedda/pi-coding-agent-go/pkg/ai"
)

// ApplyModelOverrides applies per-model and global settings overrides to a model.
func ApplyModelOverrides(m *ai.Model, settings *Settings) {
	if settings == nil {
		return
	}

	// Global BaseURL override
	if settings.BaseURL != "" && m.BaseURL == "" {
		m.BaseURL = settings.BaseURL
	}

	// Per-model overrides
	if settings.ModelOverrides == nil {
		return
	}
	override, ok := settings.ModelOverrides[m.ID]
	if !ok {
		return
	}
	if override.BaseURL != "" {
		m.BaseURL = override.BaseURL
	}
	if override.MaxOutputTokens != 0 {
		m.MaxOutputTokens = override.MaxOutputTokens
	}
	if override.ContextWindow != 0 {
		m.ContextWindow = override.ContextWindow
	}
	if len(override.CustomHeaders) > 0 {
		if m.CustomHeaders == nil {
			m.CustomHeaders = make(map[string]string)
		}
		maps.Copy(m.CustomHeaders, override.CustomHeaders)
	}
}

// ResolveModel finds a model by ID or alias.
// Checks built-in models first, then handles provider-prefixed custom models.
func ResolveModel(id string) (*ai.Model, error) {
	if id == "" {
		return &ai.ModelClaude4Sonnet, nil // Default model
	}

	// Check built-in models
	if m := ai.FindModel(id); m != nil {
		return m, nil
	}

	// Handle provider-prefixed custom models (e.g., "ollama:llama3")
	if parts := strings.SplitN(id, ":", 2); len(parts) == 2 {
		return customModel(parts[0], parts[1])
	}

	return nil, fmt.Errorf("unknown model %q", id)
}

func customModel(provider, modelID string) (*ai.Model, error) {
	var api ai.Api
	switch strings.ToLower(provider) {
	case "openai":
		api = ai.ApiOpenAI
	case "anthropic":
		api = ai.ApiAnthropic
	case "google":
		api = ai.ApiGoogle
	case "vertex":
		api = ai.ApiVertex
	case "ollama", "vllm":
		api = ai.ApiOpenAI // Ollama and vLLM use OpenAI-compatible API
	default:
		return nil, fmt.Errorf("unknown provider %q", provider)
	}

	return &ai.Model{
		ID:              modelID,
		Name:            modelID,
		Api:             api,
		MaxTokens:       128000,
		MaxOutputTokens: 16384,
		SupportsImages:  true,
		SupportsTools:   true,
	}, nil
}
