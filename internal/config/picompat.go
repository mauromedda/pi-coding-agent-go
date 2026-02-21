// ABOUTME: Compatibility layer for ~/.pi/agent/ config format (from the original pi agent)
// ABOUTME: Translates provider-centric JSON (settings.json + models.json) into pi-go Settings

package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/mauromedda/pi-coding-agent-go/pkg/ai"
)

// PiSettings maps ~/.pi/agent/settings.json.
type PiSettings struct {
	DefaultProvider      string   `json:"defaultProvider"`
	DefaultModel         string   `json:"defaultModel"`
	DefaultThinkingLevel string   `json:"defaultThinkingLevel"`
	EnabledModels        []string `json:"enabledModels"`
}

// PiModelsFile maps ~/.pi/agent/models.json.
type PiModelsFile struct {
	Providers map[string]PiProvider `json:"providers"`
}

// PiProvider describes a single provider entry in models.json.
type PiProvider struct {
	BaseURL string    `json:"baseUrl"`
	API     string    `json:"api"`
	APIKey  string    `json:"apiKey"`
	Models  []PiModel `json:"models"`
}

// PiModel describes a single model within a provider.
type PiModel struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Reasoning     bool   `json:"reasoning"`
	ContextWindow int    `json:"contextWindow"`
	MaxTokens     int    `json:"maxTokens"`
	Cost          PiCost `json:"cost"`
}

// PiCost holds per-token pricing.
type PiCost struct {
	Input      float64 `json:"input"`
	Output     float64 `json:"output"`
	CacheRead  float64 `json:"cacheRead"`
	CacheWrite float64 `json:"cacheWrite"`
}

// PiAgentDir returns ~/.pi/agent if the directory exists, empty string otherwise.
func PiAgentDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return PiAgentDirFrom(home)
}

// PiAgentDirFrom returns <home>/.pi/agent if the directory exists, empty string otherwise.
func PiAgentDirFrom(home string) string {
	dir := filepath.Join(home, ".pi", "agent")
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return ""
	}
	return dir
}

// LoadPiCompat reads settings.json + models.json from piAgentDir and returns
// translated Settings and a provider→apiKey map.
// Missing files are silently ignored (graceful degradation).
func LoadPiCompat(piAgentDir string) (*Settings, map[string]string, error) {
	settings := &Settings{}
	apiKeys := make(map[string]string)

	piSettings, settingsErr := loadPiSettings(piAgentDir)
	piModels, modelsErr := loadPiModels(piAgentDir)

	// Both missing: return empty settings
	if settingsErr != nil && modelsErr != nil {
		return settings, apiKeys, nil
	}

	// Translate model ID: "provider:model" format for ResolveModel
	if piSettings != nil && piSettings.DefaultProvider != "" && piSettings.DefaultModel != "" {
		settings.Model = piSettings.DefaultProvider + ":" + piSettings.DefaultModel
	}

	// Translate thinking level
	if piSettings != nil && piSettings.DefaultThinkingLevel != "" {
		settings.Thinking = piSettings.DefaultThinkingLevel != "off"
	}

	// Extract provider info
	if piModels != nil && piSettings != nil {
		provider, ok := piModels.Providers[piSettings.DefaultProvider]
		if ok {
			if provider.BaseURL != "" {
				settings.BaseURL = provider.BaseURL
			}

			// Find the default model's maxTokens
			for _, m := range provider.Models {
				if m.ID == piSettings.DefaultModel {
					if m.MaxTokens > 0 {
						settings.MaxTokens = m.MaxTokens
					}
					break
				}
			}
		}

		// Collect all provider API keys
		for name, p := range piModels.Providers {
			if p.APIKey != "" {
				apiKeys[name] = p.APIKey
			}
		}
	}

	return settings, apiKeys, nil
}

// MergePiAuth reads API keys from ~/.pi/agent/models.json and injects them
// into the AuthStore. Existing keys are NOT overwritten.
func MergePiAuth(store *AuthStore, piAgentDir string) {
	if piAgentDir == "" {
		return
	}

	piModels, err := loadPiModels(piAgentDir)
	if err != nil || piModels == nil {
		return
	}

	for name, p := range piModels.Providers {
		if p.APIKey == "" {
			continue
		}
		// Only set if no existing key
		existing := store.GetKey(name)
		if existing == "" {
			store.SetKey(name, p.APIKey)
		}
	}
}

// convertPiApiType maps the pi agent's API type string to an ai.Api constant.
func convertPiApiType(apiType string) ai.Api {
	switch strings.ToLower(apiType) {
	case "anthropic":
		return ai.ApiAnthropic
	case "google":
		return ai.ApiGoogle
	case "vertex":
		return ai.ApiVertex
	default:
		// "openai-completions", "openai", or anything else defaults to OpenAI
		return ai.ApiOpenAI
	}
}

func loadPiSettings(dir string) (*PiSettings, error) {
	data, err := os.ReadFile(filepath.Join(dir, "settings.json"))
	if err != nil {
		return nil, err
	}
	var s PiSettings
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

func loadPiModels(dir string) (*PiModelsFile, error) {
	data, err := os.ReadFile(filepath.Join(dir, "models.json"))
	if err != nil {
		return nil, err
	}
	var m PiModelsFile
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return &m, nil
}
