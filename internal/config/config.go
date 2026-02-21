// ABOUTME: Settings loading with global + project config deep merge
// ABOUTME: JSON-based configuration using encoding/json; no external libs

package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// Settings holds the merged configuration.
type Settings struct {
	Model       string            `json:"model,omitempty"`
	BaseURL     string            `json:"base_url,omitempty"`
	Temperature float64           `json:"temperature,omitempty"`
	MaxTokens   int               `json:"max_tokens,omitempty"`
	Yolo        bool              `json:"yolo,omitempty"`
	Thinking    bool              `json:"thinking,omitempty"`
	Env         map[string]string `json:"env,omitempty"`
}

// Load reads and merges global and project-local settings.
// Project settings override global settings.
func Load(projectRoot string) (*Settings, error) {
	global, err := loadFile(GlobalConfigFile())
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("loading global config: %w", err)
	}

	project, err := loadFile(ProjectConfigFile(projectRoot))
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("loading project config: %w", err)
	}

	merged := merge(global, project)
	return merged, nil
}

// loadFile reads a Settings from a JSON file. Returns zero Settings if file
// does not exist.
func loadFile(path string) (*Settings, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return &Settings{}, err
	}
	var s Settings
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	return &s, nil
}

// merge deep-merges project settings onto global settings.
// Non-zero project values override global values.
func merge(global, project *Settings) *Settings {
	if global == nil {
		global = &Settings{}
	}
	if project == nil {
		return global
	}

	result := *global

	if project.Model != "" {
		result.Model = project.Model
	}
	if project.BaseURL != "" {
		result.BaseURL = project.BaseURL
	}
	if project.Temperature != 0 {
		result.Temperature = project.Temperature
	}
	if project.MaxTokens != 0 {
		result.MaxTokens = project.MaxTokens
	}
	if project.Yolo {
		result.Yolo = true
	}
	if project.Thinking {
		result.Thinking = true
	}

	// Merge env maps
	if len(project.Env) > 0 {
		if result.Env == nil {
			result.Env = make(map[string]string)
		}
		for k, v := range project.Env {
			result.Env[k] = v
		}
	}

	return &result
}
