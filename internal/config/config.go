// ABOUTME: Settings loading with global + project config deep merge
// ABOUTME: JSON-based configuration using encoding/json; no external libs

package config

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"strings"
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

	// Permission rules (top-level, for backward compat)
	Allow []string `json:"allow,omitempty"`
	Deny  []string `json:"deny,omitempty"`
	Ask   []string `json:"ask,omitempty"`

	// Nested permission config (Claude Code parity)
	DefaultMode string             `json:"defaultMode,omitempty"`
	Permissions *PermissionsConfig `json:"permissions,omitempty"`

	// Status line configuration
	StatusLine *StatusLineConfig `json:"statusLine,omitempty"`

	// Hooks: event name -> list of hook definitions
	Hooks map[string][]HookDef `json:"hooks,omitempty"`

	// Sandbox configuration
	Sandbox SandboxSettings `json:"sandbox"`

	// Compaction controls auto-compaction behavior
	Compaction *CompactionSettings `json:"compaction,omitempty"`

	// Auto-compact threshold (percentage 1-100; 0 means use default 80%)
	// Deprecated: use Compaction.Enabled instead
	AutoCompactThreshold int `json:"autoCompactThreshold,omitempty"`

	// Theme name or path to a custom JSON theme file
	Theme string `json:"theme,omitempty"`

	// ModelOverrides allows per-model customization of BaseURL, headers, etc.
	ModelOverrides map[string]ModelOverride `json:"modelOverrides,omitempty"`

	// Retry controls retry behavior for API calls
	Retry *RetrySettings `json:"retry,omitempty"`

	// Terminal controls terminal rendering behavior
	Terminal *TerminalSettings `json:"terminal,omitempty"`

	// Intent configures automatic intent classification
	Intent *IntentSettings `json:"intent,omitempty"`

	// Prompts configures the versioned prompt system
	Prompts *PromptsSettings `json:"prompts,omitempty"`

	// Personality configures personality profiles and checks
	Personality *PersonalitySettings `json:"personality,omitempty"`

	// Telemetry configures cost tracking and budget alerts
	Telemetry *TelemetrySettings `json:"telemetry,omitempty"`

	// Safety configures safety guardrails
	Safety *SafetySettings `json:"safety,omitempty"`

	// Worktree configures default worktree isolation per session
	Worktree *WorktreeSettings `json:"worktree,omitempty"`

	// Minion configures the local/cloud context distillation protocol
	Minion *MinionSettings `json:"minion,omitempty"`

	// Gateway routes LLM traffic through a proxy (e.g., hikma-mirsad)
	Gateway *GatewaySettings `json:"gateway,omitempty"`
}

// ModelOverride allows per-model customization.
type ModelOverride struct {
	BaseURL          string            `json:"baseURL,omitempty"`
	CustomHeaders    map[string]string `json:"customHeaders,omitempty"`
	MaxOutputTokens  int               `json:"maxOutputTokens,omitempty"`
	ContextWindow    int               `json:"contextWindow,omitempty"`
}

// RetrySettings controls retry behavior for API calls.
type RetrySettings struct {
	MaxRetries int `json:"maxRetries,omitempty"` // default 3
	BaseDelay  int `json:"baseDelay,omitempty"`  // milliseconds; default 1000
	MaxDelay   int `json:"maxDelay,omitempty"`   // milliseconds; default 30000
}

// EffectiveMaxRetries returns MaxRetries or default (3).
func (r *RetrySettings) EffectiveMaxRetries() int {
	if r == nil || r.MaxRetries == 0 {
		return 3
	}
	return r.MaxRetries
}

// EffectiveBaseDelay returns BaseDelay or default (1000ms).
func (r *RetrySettings) EffectiveBaseDelay() int {
	if r == nil || r.BaseDelay == 0 {
		return 1000
	}
	return r.BaseDelay
}

// EffectiveMaxDelay returns MaxDelay or default (30000ms).
func (r *RetrySettings) EffectiveMaxDelay() int {
	if r == nil || r.MaxDelay == 0 {
		return 30000
	}
	return r.MaxDelay
}

// TerminalSettings controls terminal rendering.
type TerminalSettings struct {
	LineWidth int  `json:"lineWidth,omitempty"` // max line width; 0 = auto-detect
	Pager     bool `json:"pager,omitempty"`     // enable pager for long output
}

// IntentSettings configures automatic intent classification.
type IntentSettings struct {
	Enabled            *bool   `json:"enabled,omitempty"`            // nil = true
	HeuristicThreshold float64 `json:"heuristicThreshold,omitempty"` // min confidence to skip LLM; default 0.7
	AutoPlanFileCount  int     `json:"autoPlanFileCount,omitempty"`  // auto-escalate to plan if >N files; default 5
}

// IsEnabled returns whether intent classification is enabled (default true).
func (s *IntentSettings) IsEnabled() bool {
	if s == nil || s.Enabled == nil {
		return true
	}
	return *s.Enabled
}

// EffectiveHeuristicThreshold returns the threshold or default (0.7).
func (s *IntentSettings) EffectiveHeuristicThreshold() float64 {
	if s == nil || s.HeuristicThreshold == 0 {
		return 0.7
	}
	return s.HeuristicThreshold
}

// EffectiveAutoPlanFileCount returns the count or default (5).
func (s *IntentSettings) EffectiveAutoPlanFileCount() int {
	if s == nil || s.AutoPlanFileCount == 0 {
		return 5
	}
	return s.AutoPlanFileCount
}

// PromptsSettings configures the versioned prompt system.
type PromptsSettings struct {
	ActiveVersion         string `json:"activeVersion,omitempty"`         // e.g., "v1.0.0"
	OverridesDir          string `json:"overridesDir,omitempty"`          // path to overrides directory
	MaxSystemPromptTokens int    `json:"maxSystemPromptTokens,omitempty"` // budget; default 4096
}

// EffectiveMaxSystemPromptTokens returns the budget or default (4096).
func (s *PromptsSettings) EffectiveMaxSystemPromptTokens() int {
	if s == nil || s.MaxSystemPromptTokens == 0 {
		return 4096
	}
	return s.MaxSystemPromptTokens
}

// PersonalitySettings configures personality profiles and checks.
type PersonalitySettings struct {
	Profile string                      `json:"profile,omitempty"` // active profile name; default "base"
	Checks  map[string]PersonalityCheck `json:"checks,omitempty"` // per-check config
}

// EffectiveProfile returns the profile name or default ("base").
func (s *PersonalitySettings) EffectiveProfile() string {
	if s == nil || s.Profile == "" {
		return "base"
	}
	return s.Profile
}

// PersonalityCheck configures a single personality check.
type PersonalityCheck struct {
	Enabled *bool  `json:"enabled,omitempty"` // nil = true
	Level   string `json:"level,omitempty"`   // e.g., "minimal", "standard", "strict", "paranoid"
}

// IsEnabled returns whether the check is enabled (default true).
func (c PersonalityCheck) IsEnabled() bool {
	if c.Enabled == nil {
		return true
	}
	return *c.Enabled
}

// TelemetrySettings configures cost tracking and budget alerts.
type TelemetrySettings struct {
	Enabled   *bool   `json:"enabled,omitempty"`   // nil = true
	BudgetUSD float64 `json:"budgetUsd,omitempty"` // session budget limit; 0 = no limit
	WarnAtPct int     `json:"warnAtPct,omitempty"` // warn at N% of budget; default 80
}

// IsEnabled returns whether telemetry is enabled (default true).
func (s *TelemetrySettings) IsEnabled() bool {
	if s == nil || s.Enabled == nil {
		return true
	}
	return *s.Enabled
}

// EffectiveWarnAtPct returns the warning percentage or default (80).
func (s *TelemetrySettings) EffectiveWarnAtPct() int {
	if s == nil || s.WarnAtPct == 0 {
		return 80
	}
	return s.WarnAtPct
}

// SafetySettings configures safety guardrails.
type SafetySettings struct {
	NeverModify []string `json:"neverModify,omitempty"` // glob patterns for files that must never be modified
	LockedKeys  []string `json:"lockedKeys,omitempty"`  // config keys that cannot be overridden at lower levels
}

// WorktreeSettings configures default worktree isolation per session.
type WorktreeSettings struct {
	Enabled *bool `json:"enabled,omitempty"` // nil means default ON
}

// IsEnabled returns true if worktree isolation is enabled.
// Defaults to true when the setting is nil or the Enabled field is nil.
func (w *WorktreeSettings) IsEnabled() bool {
	if w == nil || w.Enabled == nil {
		return true
	}
	return *w.Enabled
}

// MinionSettings configures the minion protocol (local/cloud context distillation).
type MinionSettings struct {
	Enabled *bool  `json:"enabled,omitempty"` // nil = false (opt-in)
	Model   string `json:"model,omitempty"`   // any model ID; default "claude-haiku-4-5-20251001"
	Mode    string `json:"mode,omitempty"`    // "singular", "plural"; default "singular"
}

// IsEnabled returns whether the minion protocol is enabled (default false).
func (m *MinionSettings) IsEnabled() bool {
	if m == nil || m.Enabled == nil {
		return false
	}
	return *m.Enabled
}

// EffectiveModel returns Model or default ("claude-haiku-4-5-20251001").
func (m *MinionSettings) EffectiveModel() string {
	if m == nil || m.Model == "" {
		return "claude-haiku-4-5-20251001"
	}
	return m.Model
}

// EffectiveMode returns Mode or default ("singular").
func (m *MinionSettings) EffectiveMode() string {
	if m == nil || m.Mode == "" {
		return "singular"
	}
	return m.Mode
}

// GatewaySettings routes all LLM traffic through a proxy gateway (e.g., hikma-mirsad).
type GatewaySettings struct {
	URL   string            `json:"url"`             // e.g., "http://localhost:8080"
	Paths map[string]string `json:"paths,omitempty"` // api -> path prefix override
}

// DefaultGatewayPaths maps API types to their default gateway path prefixes.
// Includes version path segments that providers embed in their base URLs.
var DefaultGatewayPaths = map[string]string{
	"anthropic": "/anthropic",
	"openai":    "/openai",
	"google":    "/gemini/v1beta",
	"vertex":    "/vertex/v1",
}

// ResolveBaseURL returns the effective base URL for an API type.
// Returns empty string if gateway is not configured.
func (g *GatewaySettings) ResolveBaseURL(api string) string {
	if g == nil || g.URL == "" {
		return ""
	}
	path := DefaultGatewayPaths[api]
	if g.Paths != nil {
		if override, has := g.Paths[api]; has {
			path = override
		}
	}
	base := strings.TrimRight(g.URL, "/")
	if path != "" && !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return base + path
}

// PermissionsConfig holds nested permission settings (Claude Code format).
type PermissionsConfig struct {
	Allow       []string `json:"allow,omitempty"`
	Deny        []string `json:"deny,omitempty"`
	Ask         []string `json:"ask,omitempty"`
	DefaultMode string   `json:"defaultMode,omitempty"`
}

// StatusLineConfig configures the footer status line.
type StatusLineConfig struct {
	Type    string `json:"type,omitempty"`    // "command" or empty for built-in
	Command string `json:"command,omitempty"` // Shell command for external status line
	Padding int    `json:"padding,omitempty"` // Padding characters
}

// HookDef describes a lifecycle hook.
type HookDef struct {
	Matcher string `json:"matcher,omitempty"` // Tool name pattern (regex)
	Type    string `json:"type,omitempty"`    // "command"
	Command string `json:"command,omitempty"` // Shell command to run
}

// CompactionSettings controls when and how auto-compaction triggers.
type CompactionSettings struct {
	Enabled          *bool `json:"enabled,omitempty"`          // nil = true (default on)
	ReserveTokens    int   `json:"reserveTokens,omitempty"`    // tokens reserved for response (default 16384)
	KeepRecentTokens int   `json:"keepRecentTokens,omitempty"` // recent tokens to preserve (default 20000)
}

// IsEnabled returns whether compaction is enabled (default true).
func (c *CompactionSettings) IsEnabled() bool {
	if c == nil || c.Enabled == nil {
		return true
	}
	return *c.Enabled
}

// EffectiveReserveTokens returns ReserveTokens or the default (16384).
func (c *CompactionSettings) EffectiveReserveTokens() int {
	if c == nil || c.ReserveTokens == 0 {
		return 16384
	}
	return c.ReserveTokens
}

// EffectiveKeepRecentTokens returns KeepRecentTokens or the default (20000).
func (c *CompactionSettings) EffectiveKeepRecentTokens() int {
	if c == nil || c.KeepRecentTokens == 0 {
		return 20000
	}
	return c.KeepRecentTokens
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

// LoadAllWithHome reads settings from all levels with an explicit home dir.
// Level -1 (lowest priority): ~/.pi/agent/ compat layer
// Level  0: ~/.pi-go/ user settings
// Level  1: .pi-go/ project settings
// Level  2: .pi-go/settings.local.json (gitignored)
// Level  3: CLI overrides
// Level  4: Managed settings (/etc/pi-go/ or ~/Library/Application Support/)
func LoadAllWithHome(projectRoot, homeDir string, cliOverrides *Settings) (*Settings, error) {
	result := &Settings{}

	// Level -1: ~/.pi/agent/ compat (lowest priority, base layer)
	if piDir := PiAgentDirFrom(homeDir); piDir != "" {
		if piSettings, _, err := LoadPiCompat(piDir); err == nil {
			result = merge(result, piSettings)
		}
	}

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

	// Expand ${VAR} patterns in string fields
	ResolveEnvVars(result)

	return result, nil
}

// LoadAll reads settings from all five levels using the real home directory.
func LoadAll(projectRoot string, cliOverrides *Settings) (*Settings, error) {
	home, _ := os.UserHomeDir()
	return LoadAllWithHome(projectRoot, home, cliOverrides)
}

// EffectivePermissions returns the merged allow/deny/ask lists from both
// top-level and nested Permissions fields (union with dedup).
func (s *Settings) EffectivePermissions() (allow, deny, ask []string) {
	allow = append([]string{}, s.Allow...)
	deny = append([]string{}, s.Deny...)
	ask = append([]string{}, s.Ask...)

	if s.Permissions != nil {
		allow = dedupStrings(allow, s.Permissions.Allow)
		deny = dedupStrings(deny, s.Permissions.Deny)
		ask = dedupStrings(ask, s.Permissions.Ask)
	}
	return allow, deny, ask
}

// EffectiveDefaultMode returns the effective default permission mode.
// Nested Permissions.DefaultMode takes precedence over top-level DefaultMode.
func (s *Settings) EffectiveDefaultMode() string {
	if s.Permissions != nil && s.Permissions.DefaultMode != "" {
		return s.Permissions.DefaultMode
	}
	return s.DefaultMode
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
	if project.DefaultMode != "" {
		result.DefaultMode = project.DefaultMode
	}

	// Merge env maps
	if len(project.Env) > 0 {
		if result.Env == nil {
			result.Env = make(map[string]string)
		}
		maps.Copy(result.Env, project.Env)
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

	// Permissions: merge nested config
	if project.Permissions != nil {
		if result.Permissions == nil {
			result.Permissions = &PermissionsConfig{}
		}
		if len(project.Permissions.Allow) > 0 {
			result.Permissions.Allow = dedupStrings(result.Permissions.Allow, project.Permissions.Allow)
		}
		if len(project.Permissions.Deny) > 0 {
			result.Permissions.Deny = dedupStrings(result.Permissions.Deny, project.Permissions.Deny)
		}
		if len(project.Permissions.Ask) > 0 {
			result.Permissions.Ask = dedupStrings(result.Permissions.Ask, project.Permissions.Ask)
		}
		if project.Permissions.DefaultMode != "" {
			result.Permissions.DefaultMode = project.Permissions.DefaultMode
		}
	}

	// StatusLine: override if present
	if project.StatusLine != nil {
		result.StatusLine = project.StatusLine
	}

	// Hooks: merge by event name
	if len(project.Hooks) > 0 {
		if result.Hooks == nil {
			result.Hooks = make(map[string][]HookDef)
		}
		maps.Copy(result.Hooks, project.Hooks)
	}

	// AutoCompactThreshold: override if non-zero
	if project.AutoCompactThreshold != 0 {
		result.AutoCompactThreshold = project.AutoCompactThreshold
	}

	// Compaction: merge if present
	if project.Compaction != nil {
		if result.Compaction == nil {
			result.Compaction = &CompactionSettings{}
		}
		if project.Compaction.Enabled != nil {
			result.Compaction.Enabled = project.Compaction.Enabled
		}
		if project.Compaction.ReserveTokens != 0 {
			result.Compaction.ReserveTokens = project.Compaction.ReserveTokens
		}
		if project.Compaction.KeepRecentTokens != 0 {
			result.Compaction.KeepRecentTokens = project.Compaction.KeepRecentTokens
		}
	}

	// Sandbox: override if present
	if len(project.Sandbox.ExcludedCommands) > 0 {
		result.Sandbox.ExcludedCommands = project.Sandbox.ExcludedCommands
	}
	if len(project.Sandbox.AllowedDomains) > 0 {
		result.Sandbox.AllowedDomains = project.Sandbox.AllowedDomains
	}

	// ModelOverrides: merge by model ID
	if len(project.ModelOverrides) > 0 {
		if result.ModelOverrides == nil {
			result.ModelOverrides = make(map[string]ModelOverride)
		}
		maps.Copy(result.ModelOverrides, project.ModelOverrides)
	}

	// Retry: override if present
	if project.Retry != nil {
		if result.Retry == nil {
			result.Retry = &RetrySettings{}
		}
		if project.Retry.MaxRetries != 0 {
			result.Retry.MaxRetries = project.Retry.MaxRetries
		}
		if project.Retry.BaseDelay != 0 {
			result.Retry.BaseDelay = project.Retry.BaseDelay
		}
		if project.Retry.MaxDelay != 0 {
			result.Retry.MaxDelay = project.Retry.MaxDelay
		}
	}

	// Terminal: override if present
	if project.Terminal != nil {
		result.Terminal = project.Terminal
	}

	// Intent: merge if present
	if project.Intent != nil {
		if result.Intent == nil {
			result.Intent = &IntentSettings{}
		}
		if project.Intent.Enabled != nil {
			result.Intent.Enabled = project.Intent.Enabled
		}
		if project.Intent.HeuristicThreshold != 0 {
			result.Intent.HeuristicThreshold = project.Intent.HeuristicThreshold
		}
		if project.Intent.AutoPlanFileCount != 0 {
			result.Intent.AutoPlanFileCount = project.Intent.AutoPlanFileCount
		}
	}

	// Prompts: merge if present
	if project.Prompts != nil {
		if result.Prompts == nil {
			result.Prompts = &PromptsSettings{}
		}
		if project.Prompts.ActiveVersion != "" {
			result.Prompts.ActiveVersion = project.Prompts.ActiveVersion
		}
		if project.Prompts.OverridesDir != "" {
			result.Prompts.OverridesDir = project.Prompts.OverridesDir
		}
		if project.Prompts.MaxSystemPromptTokens != 0 {
			result.Prompts.MaxSystemPromptTokens = project.Prompts.MaxSystemPromptTokens
		}
	}

	// Personality: merge if present
	if project.Personality != nil {
		if result.Personality == nil {
			result.Personality = &PersonalitySettings{}
		}
		if project.Personality.Profile != "" {
			result.Personality.Profile = project.Personality.Profile
		}
		if len(project.Personality.Checks) > 0 {
			if result.Personality.Checks == nil {
				result.Personality.Checks = make(map[string]PersonalityCheck)
			}
			maps.Copy(result.Personality.Checks, project.Personality.Checks)
		}
	}

	// Telemetry: merge if present
	if project.Telemetry != nil {
		if result.Telemetry == nil {
			result.Telemetry = &TelemetrySettings{}
		} else {
			t := *result.Telemetry
			result.Telemetry = &t
		}
		if project.Telemetry.Enabled != nil {
			result.Telemetry.Enabled = project.Telemetry.Enabled
		}
		if project.Telemetry.BudgetUSD != 0 {
			result.Telemetry.BudgetUSD = project.Telemetry.BudgetUSD
		}
		if project.Telemetry.WarnAtPct != 0 {
			result.Telemetry.WarnAtPct = project.Telemetry.WarnAtPct
		}
	}

	// Safety: merge if present
	if project.Safety != nil {
		if result.Safety == nil {
			result.Safety = &SafetySettings{}
		} else {
			s := *result.Safety
			result.Safety = &s
		}
		if len(project.Safety.NeverModify) > 0 {
			result.Safety.NeverModify = dedupStrings(result.Safety.NeverModify, project.Safety.NeverModify)
		}
		if len(project.Safety.LockedKeys) > 0 {
			result.Safety.LockedKeys = dedupStrings(result.Safety.LockedKeys, project.Safety.LockedKeys)
		}
	}

	// Worktree: merge if present
	if project.Worktree != nil {
		if result.Worktree == nil {
			result.Worktree = &WorktreeSettings{}
		} else {
			w := *result.Worktree
			result.Worktree = &w
		}
		if project.Worktree.Enabled != nil {
			result.Worktree.Enabled = project.Worktree.Enabled
		}
	}

	// Minion: merge if present
	if project.Minion != nil {
		if result.Minion == nil {
			result.Minion = &MinionSettings{}
		} else {
			m := *result.Minion
			result.Minion = &m
		}
		if project.Minion.Enabled != nil {
			result.Minion.Enabled = project.Minion.Enabled
		}
		if project.Minion.Model != "" {
			result.Minion.Model = project.Minion.Model
		}
		if project.Minion.Mode != "" {
			result.Minion.Mode = project.Minion.Mode
		}
	}

	// Gateway: merge if present (deep-copy to prevent mutation)
	if project.Gateway != nil {
		if result.Gateway == nil {
			result.Gateway = &GatewaySettings{}
		} else {
			g := *result.Gateway
			if result.Gateway.Paths != nil {
				g.Paths = make(map[string]string, len(result.Gateway.Paths))
				maps.Copy(g.Paths, result.Gateway.Paths)
			}
			result.Gateway = &g
		}
		if project.Gateway.URL != "" {
			result.Gateway.URL = project.Gateway.URL
		}
		if project.Gateway.Paths != nil {
			if result.Gateway.Paths == nil {
				result.Gateway.Paths = make(map[string]string)
			}
			maps.Copy(result.Gateway.Paths, project.Gateway.Paths)
		}
	} else if result.Gateway != nil {
		// Deep-copy global gateway to avoid shared map mutation
		g := *result.Gateway
		if result.Gateway.Paths != nil {
			g.Paths = make(map[string]string, len(result.Gateway.Paths))
			maps.Copy(g.Paths, result.Gateway.Paths)
		}
		result.Gateway = &g
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
