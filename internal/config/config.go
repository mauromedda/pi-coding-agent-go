// ABOUTME: Settings loading with global + project config deep merge
// ABOUTME: JSON-based configuration using encoding/json; no external libs

package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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

	// Permission rules
	Allow []string `json:"allow,omitempty"`
	Deny  []string `json:"deny,omitempty"`
	Ask   []string `json:"ask,omitempty"`

	// Hooks: event name -> list of hook definitions
	Hooks map[string][]HookDef `json:"hooks,omitempty"`

	// Sandbox configuration
	Sandbox SandboxSettings `json:"sandbox,omitempty"`
}

// HookDef describes a lifecycle hook.
type HookDef struct {
	Matcher string `json:"matcher,omitempty"` // Tool name pattern (regex)
	Type    string `json:"type,omitempty"`    // "command"
	Command string `json:"command,omitempty"` // Shell command to run
}

// SandboxSettings configures the OS sandbox.
type SandboxSettings struct {
	ExcludedCommands []string `json:"excludedCommands,omitempty"`
	AllowedDomains   []string `json:"allowedDomains,omitempty"`
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

// SettingsLevel represents the precedence level of a settings source.
type SettingsLevel int

const (
	LevelUser    SettingsLevel = iota // ~/.pi-go/settings.json
	LevelProject                      // .pi-go/settings.json
	LevelLocal                        // .pi-go/settings.local.json (gitignored)
	LevelCLI                          // Command-line overrides
	LevelManaged                      // /Library/Application Support/pi-go/ or /etc/pi-go/
)

// LoadAllWithHome reads settings from all five levels with an explicit home dir.
func LoadAllWithHome(projectRoot, homeDir string, cliOverrides *Settings) (*Settings, error) {
	result := &Settings{}

	// Level 0: User settings (old config.json + new settings.json)
	sources := []string{
		filepath.Join(homeDir, ".pi-go", "config.json"),
		filepath.Join(homeDir, ".pi-go", "settings.json"),
	}
	for _, path := range sources {
		if s, err := loadFile(path); err == nil {
			result = merge(result, s)
		}
	}

	// Level 1: Project settings
	projectSources := []string{
		filepath.Join(projectRoot, ".pi-go", "config.json"),
		filepath.Join(projectRoot, ".pi-go", "settings.json"),
	}
	for _, path := range projectSources {
		if s, err := loadFile(path); err == nil {
			result = merge(result, s)
		}
	}

	// Level 2: Local settings (gitignored)
	localPath := filepath.Join(projectRoot, ".pi-go", "settings.local.json")
	if s, err := loadFile(localPath); err == nil {
		result = merge(result, s)
	}

	// Level 3: CLI overrides
	if cliOverrides != nil {
		result = merge(result, cliOverrides)
	}

	// Level 4: Managed settings (enterprise/system)
	managedPath := ManagedSettingsFile()
	if s, err := loadFile(managedPath); err == nil {
		result = merge(result, s)
	}

	return result, nil
}

// LoadAll reads settings from all five levels using the real home directory.
func LoadAll(projectRoot string, cliOverrides *Settings) (*Settings, error) {
	home, _ := os.UserHomeDir()
	return LoadAllWithHome(projectRoot, home, cliOverrides)
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

	// Permission rules: union with dedup
	if len(project.Allow) > 0 {
		result.Allow = dedupStrings(result.Allow, project.Allow)
	}
	if len(project.Deny) > 0 {
		result.Deny = dedupStrings(result.Deny, project.Deny)
	}
	if len(project.Ask) > 0 {
		result.Ask = dedupStrings(result.Ask, project.Ask)
	}

	// Hooks: merge by event name
	if len(project.Hooks) > 0 {
		if result.Hooks == nil {
			result.Hooks = make(map[string][]HookDef)
		}
		for event, hooks := range project.Hooks {
			result.Hooks[event] = hooks
		}
	}

	// Sandbox: override if present
	if len(project.Sandbox.ExcludedCommands) > 0 {
		result.Sandbox.ExcludedCommands = project.Sandbox.ExcludedCommands
	}
	if len(project.Sandbox.AllowedDomains) > 0 {
		result.Sandbox.AllowedDomains = project.Sandbox.AllowedDomains
	}

	return &result
}

// dedupStrings returns the union of a and b with duplicates removed.
// Order: a first, then new elements from b.
func dedupStrings(a, b []string) []string {
	seen := make(map[string]bool, len(a)+len(b))
	result := make([]string, 0, len(a)+len(b))
	for _, s := range a {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	for _, s := range b {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}
