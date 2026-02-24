// ABOUTME: Tests for config loading, merging, and auth storage
// ABOUTME: Uses temp directories for isolated file-based tests

package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestMerge(t *testing.T) {
	t.Parallel()

	global := &Settings{Model: "default-model", Temperature: 0.7}
	project := &Settings{Model: "project-model"}

	result := merge(global, project)

	if result.Model != "project-model" {
		t.Errorf("Model = %q, want %q", result.Model, "project-model")
	}
	if result.Temperature != 0.7 {
		t.Errorf("Temperature = %f, want 0.7", result.Temperature)
	}
}

func TestMerge_Nil(t *testing.T) {
	t.Parallel()

	result := merge(nil, nil)
	if result == nil {
		t.Fatal("merge(nil, nil) should return non-nil")
	}
}

func TestMerge_EnvMerge(t *testing.T) {
	t.Parallel()

	global := &Settings{Env: map[string]string{"A": "1", "B": "2"}}
	project := &Settings{Env: map[string]string{"B": "override", "C": "3"}}

	result := merge(global, project)

	if result.Env["A"] != "1" {
		t.Error("expected A=1 from global")
	}
	if result.Env["B"] != "override" {
		t.Error("expected B=override from project")
	}
	if result.Env["C"] != "3" {
		t.Error("expected C=3 from project")
	}
}

func TestLoadFile_NotExist(t *testing.T) {
	t.Parallel()

	s, err := loadFile("/nonexistent/path/config.json")
	if !os.IsNotExist(err) {
		t.Errorf("expected not exist error, got %v", err)
	}
	if s == nil {
		t.Error("expected non-nil default settings")
	}
}

func TestLoadFile_ValidJSON(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte(`{"model":"test"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	s, err := loadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if s.Model != "test" {
		t.Errorf("Model = %q, want %q", s.Model, "test")
	}
}

func TestMerge_DenyUnion(t *testing.T) {
	t.Parallel()

	global := &Settings{
		Allow: []string{"read", "grep"},
		Deny:  []string{"rm*"},
		Ask:   []string{"bash"},
	}
	project := &Settings{
		Allow: []string{"write"},
		Deny:  []string{"eval"},
		Ask:   []string{"exec"},
	}

	result := merge(global, project)

	// Deny should be unioned: both "rm*" and "eval" present
	if !containsAll(result.Deny, "rm*", "eval") {
		t.Errorf("Deny = %v, want both rm* and eval", result.Deny)
	}
	// Allow should be unioned
	if !containsAll(result.Allow, "read", "grep", "write") {
		t.Errorf("Allow = %v, want read, grep, write", result.Allow)
	}
	// Ask should be unioned
	if !containsAll(result.Ask, "bash", "exec") {
		t.Errorf("Ask = %v, want bash and exec", result.Ask)
	}
}

func TestMerge_DenyUnion_Dedup(t *testing.T) {
	t.Parallel()

	global := &Settings{Deny: []string{"rm*", "eval"}}
	project := &Settings{Deny: []string{"eval", "curl"}}

	result := merge(global, project)

	// Should deduplicate: "eval" only once
	count := 0
	for _, d := range result.Deny {
		if d == "eval" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("Deny has %d instances of 'eval', want 1: %v", count, result.Deny)
	}
	if len(result.Deny) != 3 {
		t.Errorf("Deny length = %d, want 3 (rm*, eval, curl): %v", len(result.Deny), result.Deny)
	}
}

func containsAll(slice []string, items ...string) bool {
	set := make(map[string]bool, len(slice))
	for _, s := range slice {
		set[s] = true
	}
	for _, item := range items {
		if !set[item] {
			return false
		}
	}
	return true
}

func TestSettings_EffectivePermissions_Merge(t *testing.T) {
	t.Parallel()

	s := &Settings{
		Allow: []string{"read", "grep"},
		Deny:  []string{"rm*"},
		Ask:   []string{"bash"},
		Permissions: &PermissionsConfig{
			Allow: []string{"write", "read"}, // "read" is a dup
			Deny:  []string{"eval"},
			Ask:   []string{"exec"},
		},
	}

	allow, deny, ask := s.EffectivePermissions()

	// Union with dedup
	if !containsAll(allow, "read", "grep", "write") {
		t.Errorf("allow = %v, want read, grep, write", allow)
	}
	// "read" should not appear twice
	count := 0
	for _, a := range allow {
		if a == "read" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("allow has %d instances of 'read', want 1", count)
	}

	if !containsAll(deny, "rm*", "eval") {
		t.Errorf("deny = %v, want rm*, eval", deny)
	}
	if !containsAll(ask, "bash", "exec") {
		t.Errorf("ask = %v, want bash, exec", ask)
	}
}

func TestSettings_EffectivePermissions_NilPermissions(t *testing.T) {
	t.Parallel()

	s := &Settings{
		Allow: []string{"read"},
		Deny:  []string{"rm*"},
	}

	allow, deny, ask := s.EffectivePermissions()

	if !containsAll(allow, "read") || len(allow) != 1 {
		t.Errorf("allow = %v, want [read]", allow)
	}
	if !containsAll(deny, "rm*") || len(deny) != 1 {
		t.Errorf("deny = %v, want [rm*]", deny)
	}
	if len(ask) != 0 {
		t.Errorf("ask = %v, want empty", ask)
	}
}

func TestSettings_EffectiveDefaultMode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		defaultMode string
		permissions *PermissionsConfig
		want        string
	}{
		{"empty", "", nil, ""},
		{"top-level only", "acceptEdits", nil, "acceptEdits"},
		{"nested only", "", &PermissionsConfig{DefaultMode: "dontAsk"}, "dontAsk"},
		{"nested overrides top-level", "acceptEdits", &PermissionsConfig{DefaultMode: "dontAsk"}, "dontAsk"},
		{"nested empty falls back to top-level", "acceptEdits", &PermissionsConfig{}, "acceptEdits"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Settings{
				DefaultMode: tt.defaultMode,
				Permissions: tt.permissions,
			}
			if got := s.EffectiveDefaultMode(); got != tt.want {
				t.Errorf("EffectiveDefaultMode() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMerge_DefaultMode(t *testing.T) {
	t.Parallel()

	global := &Settings{DefaultMode: "acceptEdits"}
	project := &Settings{DefaultMode: "dontAsk"}

	result := merge(global, project)
	if result.DefaultMode != "dontAsk" {
		t.Errorf("DefaultMode = %q, want %q", result.DefaultMode, "dontAsk")
	}
}

func TestMerge_StatusLine(t *testing.T) {
	t.Parallel()

	global := &Settings{}
	project := &Settings{
		StatusLine: &StatusLineConfig{
			Type:    "command",
			Command: "echo hello",
			Padding: 2,
		},
	}

	result := merge(global, project)
	if result.StatusLine == nil {
		t.Fatal("StatusLine should be set")
	}
	if result.StatusLine.Command != "echo hello" {
		t.Errorf("StatusLine.Command = %q, want %q", result.StatusLine.Command, "echo hello")
	}
	if result.StatusLine.Padding != 2 {
		t.Errorf("StatusLine.Padding = %d, want 2", result.StatusLine.Padding)
	}
}

func TestMerge_Permissions(t *testing.T) {
	t.Parallel()

	global := &Settings{
		Permissions: &PermissionsConfig{
			Allow: []string{"read"},
			Deny:  []string{"rm*"},
		},
	}
	project := &Settings{
		Permissions: &PermissionsConfig{
			Allow:       []string{"write"},
			DefaultMode: "dontAsk",
		},
	}

	result := merge(global, project)
	if result.Permissions == nil {
		t.Fatal("Permissions should be set")
	}
	if !containsAll(result.Permissions.Allow, "read", "write") {
		t.Errorf("Permissions.Allow = %v, want read, write", result.Permissions.Allow)
	}
	if !containsAll(result.Permissions.Deny, "rm*") {
		t.Errorf("Permissions.Deny = %v, want rm*", result.Permissions.Deny)
	}
	if result.Permissions.DefaultMode != "dontAsk" {
		t.Errorf("Permissions.DefaultMode = %q, want %q", result.Permissions.DefaultMode, "dontAsk")
	}
}

func TestLoadFile_WithNewFields(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	data := `{
		"defaultMode": "acceptEdits",
		"permissions": {
			"allow": ["bash"],
			"deny": ["rm*"],
			"defaultMode": "dontAsk"
		},
		"statusLine": {
			"type": "command",
			"command": "echo test",
			"padding": 3
		}
	}`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}

	s, err := loadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if s.DefaultMode != "acceptEdits" {
		t.Errorf("DefaultMode = %q, want %q", s.DefaultMode, "acceptEdits")
	}
	if s.Permissions == nil {
		t.Fatal("Permissions should be set")
	}
	if s.Permissions.DefaultMode != "dontAsk" {
		t.Errorf("Permissions.DefaultMode = %q, want %q", s.Permissions.DefaultMode, "dontAsk")
	}
	if s.StatusLine == nil {
		t.Fatal("StatusLine should be set")
	}
	if s.StatusLine.Command != "echo test" {
		t.Errorf("StatusLine.Command = %q, want %q", s.StatusLine.Command, "echo test")
	}
}

func TestMerge_AutoCompactThreshold(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		global  *Settings
		project *Settings
		want    int
	}{
		{
			"project overrides global",
			&Settings{AutoCompactThreshold: 80},
			&Settings{AutoCompactThreshold: 50},
			50,
		},
		{
			"global preserved when project is zero",
			&Settings{AutoCompactThreshold: 70},
			&Settings{},
			70,
		},
		{
			"both zero",
			&Settings{},
			&Settings{},
			0,
		},
		{
			"project sets when global is zero",
			&Settings{},
			&Settings{AutoCompactThreshold: 60},
			60,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := merge(tt.global, tt.project)
			if result.AutoCompactThreshold != tt.want {
				t.Errorf("AutoCompactThreshold = %d, want %d", result.AutoCompactThreshold, tt.want)
			}
		})
	}
}

func TestLoadFile_AutoCompactThreshold(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	data := `{"autoCompactThreshold": 60}`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}

	s, err := loadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if s.AutoCompactThreshold != 60 {
		t.Errorf("AutoCompactThreshold = %d, want 60", s.AutoCompactThreshold)
	}
}

func TestAuthStore_GetKey_EnvFallback(t *testing.T) {
	store := &AuthStore{Keys: make(map[string]string)}

	t.Setenv("PI_API_KEY_ANTHROPIC", "test-key-123")

	got := store.GetKey("ANTHROPIC")
	if got != "test-key-123" {
		t.Errorf("GetKey(ANTHROPIC) = %q, want %q", got, "test-key-123")
	}
}

func TestAuthStore_SetAndGet(t *testing.T) {
	t.Parallel()

	store := &AuthStore{Keys: make(map[string]string)}
	store.SetKey("openai", "sk-test")

	got := store.GetKey("openai")
	if got != "sk-test" {
		t.Errorf("GetKey = %q, want %q", got, "sk-test")
	}
}

func TestAuthStore_GetKey_CaseNormalization(t *testing.T) {
	store := &AuthStore{Keys: make(map[string]string)}

	// Set env var with uppercase; query with lowercase provider.
	t.Setenv("PI_API_KEY_OPENAI", "from-env")

	got := store.GetKey("openai")
	if got != "from-env" {
		t.Errorf("GetKey(openai) = %q, want %q (should normalize to uppercase)", got, "from-env")
	}
}

func TestCompactionSettings_Defaults(t *testing.T) {
	t.Parallel()

	var cs *CompactionSettings // nil

	if !cs.IsEnabled() {
		t.Error("nil CompactionSettings should be enabled by default")
	}
	if cs.EffectiveReserveTokens() != 16384 {
		t.Errorf("EffectiveReserveTokens = %d, want 16384", cs.EffectiveReserveTokens())
	}
	if cs.EffectiveKeepRecentTokens() != 20000 {
		t.Errorf("EffectiveKeepRecentTokens = %d, want 20000", cs.EffectiveKeepRecentTokens())
	}
}

func TestCompactionSettings_CustomValues(t *testing.T) {
	t.Parallel()

	f := false
	cs := &CompactionSettings{
		Enabled:          &f,
		ReserveTokens:    8192,
		KeepRecentTokens: 10000,
	}

	if cs.IsEnabled() {
		t.Error("should be disabled when Enabled=false")
	}
	if cs.EffectiveReserveTokens() != 8192 {
		t.Errorf("EffectiveReserveTokens = %d, want 8192", cs.EffectiveReserveTokens())
	}
	if cs.EffectiveKeepRecentTokens() != 10000 {
		t.Errorf("EffectiveKeepRecentTokens = %d, want 10000", cs.EffectiveKeepRecentTokens())
	}
}

func TestMerge_Compaction(t *testing.T) {
	t.Parallel()

	f := false
	global := &Settings{
		Compaction: &CompactionSettings{ReserveTokens: 16384},
	}
	project := &Settings{
		Compaction: &CompactionSettings{
			Enabled:          &f,
			KeepRecentTokens: 5000,
		},
	}

	result := merge(global, project)

	if result.Compaction == nil {
		t.Fatal("Compaction should be set")
	}
	if result.Compaction.IsEnabled() {
		t.Error("project should override Enabled to false")
	}
	if result.Compaction.ReserveTokens != 16384 {
		t.Errorf("ReserveTokens = %d, want 16384 (from global)", result.Compaction.ReserveTokens)
	}
	if result.Compaction.KeepRecentTokens != 5000 {
		t.Errorf("KeepRecentTokens = %d, want 5000 (from project)", result.Compaction.KeepRecentTokens)
	}
}

func TestRetrySettings_Defaults(t *testing.T) {
	t.Parallel()

	var rs *RetrySettings // nil
	if rs.EffectiveMaxRetries() != 3 {
		t.Errorf("EffectiveMaxRetries = %d, want 3", rs.EffectiveMaxRetries())
	}
	if rs.EffectiveBaseDelay() != 1000 {
		t.Errorf("EffectiveBaseDelay = %d, want 1000", rs.EffectiveBaseDelay())
	}
	if rs.EffectiveMaxDelay() != 30000 {
		t.Errorf("EffectiveMaxDelay = %d, want 30000", rs.EffectiveMaxDelay())
	}
}

func TestRetrySettings_Custom(t *testing.T) {
	t.Parallel()

	rs := &RetrySettings{MaxRetries: 5, BaseDelay: 500, MaxDelay: 60000}
	if rs.EffectiveMaxRetries() != 5 {
		t.Errorf("EffectiveMaxRetries = %d, want 5", rs.EffectiveMaxRetries())
	}
	if rs.EffectiveBaseDelay() != 500 {
		t.Errorf("EffectiveBaseDelay = %d, want 500", rs.EffectiveBaseDelay())
	}
}

func TestMerge_RetrySettings(t *testing.T) {
	t.Parallel()

	global := &Settings{Retry: &RetrySettings{MaxRetries: 3, BaseDelay: 1000}}
	project := &Settings{Retry: &RetrySettings{MaxRetries: 5}}

	result := merge(global, project)
	if result.Retry == nil {
		t.Fatal("Retry should be set")
	}
	if result.Retry.MaxRetries != 5 {
		t.Errorf("MaxRetries = %d, want 5 (from project)", result.Retry.MaxRetries)
	}
	if result.Retry.BaseDelay != 1000 {
		t.Errorf("BaseDelay = %d, want 1000 (from global)", result.Retry.BaseDelay)
	}
}

func TestMerge_Terminal(t *testing.T) {
	t.Parallel()

	global := &Settings{}
	project := &Settings{Terminal: &TerminalSettings{LineWidth: 120, Pager: true}}

	result := merge(global, project)
	if result.Terminal == nil {
		t.Fatal("Terminal should be set")
	}
	if result.Terminal.LineWidth != 120 {
		t.Errorf("LineWidth = %d, want 120", result.Terminal.LineWidth)
	}
	if !result.Terminal.Pager {
		t.Error("Pager should be true")
	}
}

func TestMerge_ModelOverrides(t *testing.T) {
	t.Parallel()

	global := &Settings{
		ModelOverrides: map[string]ModelOverride{
			"model-a": {BaseURL: "https://a.example.com"},
		},
	}
	project := &Settings{
		ModelOverrides: map[string]ModelOverride{
			"model-b": {MaxOutputTokens: 8192},
		},
	}

	result := merge(global, project)
	if len(result.ModelOverrides) != 2 {
		t.Errorf("ModelOverrides length = %d, want 2", len(result.ModelOverrides))
	}
	if result.ModelOverrides["model-a"].BaseURL != "https://a.example.com" {
		t.Error("model-a override should be preserved from global")
	}
	if result.ModelOverrides["model-b"].MaxOutputTokens != 8192 {
		t.Error("model-b override should be set from project")
	}
}

func TestLoadFile_CompactionSettings(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	data := `{"compaction":{"enabled":false,"reserveTokens":8192,"keepRecentTokens":10000}}`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}

	s, err := loadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if s.Compaction == nil {
		t.Fatal("Compaction should be set")
	}
	if s.Compaction.IsEnabled() {
		t.Error("should be disabled")
	}
	if s.Compaction.ReserveTokens != 8192 {
		t.Errorf("ReserveTokens = %d, want 8192", s.Compaction.ReserveTokens)
	}
}

func TestIntentSettings_Defaults(t *testing.T) {
	t.Parallel()

	var is *IntentSettings // nil

	if !is.IsEnabled() {
		t.Error("nil IntentSettings should be enabled by default")
	}
	if is.EffectiveHeuristicThreshold() != 0.7 {
		t.Errorf("EffectiveHeuristicThreshold = %f, want 0.7", is.EffectiveHeuristicThreshold())
	}
	if is.EffectiveAutoPlanFileCount() != 5 {
		t.Errorf("EffectiveAutoPlanFileCount = %d, want 5", is.EffectiveAutoPlanFileCount())
	}
}

func TestIntentSettings_CustomValues(t *testing.T) {
	t.Parallel()

	f := false
	is := &IntentSettings{
		Enabled:            &f,
		HeuristicThreshold: 0.9,
		AutoPlanFileCount:  10,
	}

	if is.IsEnabled() {
		t.Error("should be disabled when Enabled=false")
	}
	if is.EffectiveHeuristicThreshold() != 0.9 {
		t.Errorf("EffectiveHeuristicThreshold = %f, want 0.9", is.EffectiveHeuristicThreshold())
	}
	if is.EffectiveAutoPlanFileCount() != 10 {
		t.Errorf("EffectiveAutoPlanFileCount = %d, want 10", is.EffectiveAutoPlanFileCount())
	}
}

func TestPromptsSettings_Defaults(t *testing.T) {
	t.Parallel()

	var ps *PromptsSettings // nil

	if ps.EffectiveMaxSystemPromptTokens() != 4096 {
		t.Errorf("EffectiveMaxSystemPromptTokens = %d, want 4096", ps.EffectiveMaxSystemPromptTokens())
	}
}

func TestPromptsSettings_CustomValues(t *testing.T) {
	t.Parallel()

	ps := &PromptsSettings{
		ActiveVersion:         "v2.0.0",
		OverridesDir:          "/custom/prompts",
		MaxSystemPromptTokens: 8192,
	}

	if ps.EffectiveMaxSystemPromptTokens() != 8192 {
		t.Errorf("EffectiveMaxSystemPromptTokens = %d, want 8192", ps.EffectiveMaxSystemPromptTokens())
	}
}

func TestPersonalitySettings_Defaults(t *testing.T) {
	t.Parallel()

	var ps *PersonalitySettings // nil

	if ps.EffectiveProfile() != "base" {
		t.Errorf("EffectiveProfile = %q, want %q", ps.EffectiveProfile(), "base")
	}
}

func TestPersonalitySettings_CustomValues(t *testing.T) {
	t.Parallel()

	ps := &PersonalitySettings{
		Profile: "concise",
		Checks: map[string]PersonalityCheck{
			"humor": {Level: "minimal"},
		},
	}

	if ps.EffectiveProfile() != "concise" {
		t.Errorf("EffectiveProfile = %q, want %q", ps.EffectiveProfile(), "concise")
	}
}

func TestPersonalityCheck_IsEnabled(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		check PersonalityCheck
		want  bool
	}{
		{"nil enabled defaults to true", PersonalityCheck{}, true},
		{"enabled true", PersonalityCheck{Enabled: new(true)}, true},
		{"enabled false", PersonalityCheck{Enabled: new(false)}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.check.IsEnabled(); got != tt.want {
				t.Errorf("IsEnabled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTelemetrySettings_Defaults(t *testing.T) {
	t.Parallel()

	var ts *TelemetrySettings // nil

	if !ts.IsEnabled() {
		t.Error("nil TelemetrySettings should be enabled by default")
	}
	if ts.EffectiveWarnAtPct() != 80 {
		t.Errorf("EffectiveWarnAtPct = %d, want 80", ts.EffectiveWarnAtPct())
	}
}

func TestTelemetrySettings_CustomValues(t *testing.T) {
	t.Parallel()

	f := false
	ts := &TelemetrySettings{
		Enabled:   &f,
		BudgetUSD: 25.0,
		WarnAtPct: 90,
	}

	if ts.IsEnabled() {
		t.Error("should be disabled when Enabled=false")
	}
	if ts.EffectiveWarnAtPct() != 90 {
		t.Errorf("EffectiveWarnAtPct = %d, want 90", ts.EffectiveWarnAtPct())
	}
}

func TestMerge_IntentSettings(t *testing.T) {
	t.Parallel()

	f := false
	global := &Settings{
		Intent: &IntentSettings{HeuristicThreshold: 0.7, AutoPlanFileCount: 5},
	}
	project := &Settings{
		Intent: &IntentSettings{Enabled: &f, AutoPlanFileCount: 10},
	}

	result := merge(global, project)

	if result.Intent == nil {
		t.Fatal("Intent should be set")
	}
	if result.Intent.IsEnabled() {
		t.Error("project should override Enabled to false")
	}
	if result.Intent.HeuristicThreshold != 0.7 {
		t.Errorf("HeuristicThreshold = %f, want 0.7 (from global)", result.Intent.HeuristicThreshold)
	}
	if result.Intent.AutoPlanFileCount != 10 {
		t.Errorf("AutoPlanFileCount = %d, want 10 (from project)", result.Intent.AutoPlanFileCount)
	}
}

func TestMerge_PromptsSettings(t *testing.T) {
	t.Parallel()

	global := &Settings{
		Prompts: &PromptsSettings{ActiveVersion: "v1.0.0", MaxSystemPromptTokens: 4096},
	}
	project := &Settings{
		Prompts: &PromptsSettings{OverridesDir: "/custom/prompts"},
	}

	result := merge(global, project)

	if result.Prompts == nil {
		t.Fatal("Prompts should be set")
	}
	if result.Prompts.ActiveVersion != "v1.0.0" {
		t.Errorf("ActiveVersion = %q, want %q (from global)", result.Prompts.ActiveVersion, "v1.0.0")
	}
	if result.Prompts.OverridesDir != "/custom/prompts" {
		t.Errorf("OverridesDir = %q, want %q (from project)", result.Prompts.OverridesDir, "/custom/prompts")
	}
	if result.Prompts.MaxSystemPromptTokens != 4096 {
		t.Errorf("MaxSystemPromptTokens = %d, want 4096 (from global)", result.Prompts.MaxSystemPromptTokens)
	}
}

func TestMerge_PersonalitySettings(t *testing.T) {
	t.Parallel()

	global := &Settings{
		Personality: &PersonalitySettings{
			Profile: "base",
			Checks: map[string]PersonalityCheck{
				"humor": {Level: "standard"},
			},
		},
	}
	project := &Settings{
		Personality: &PersonalitySettings{
			Profile: "concise",
			Checks: map[string]PersonalityCheck{
				"verbosity": {Level: "minimal"},
			},
		},
	}

	result := merge(global, project)

	if result.Personality == nil {
		t.Fatal("Personality should be set")
	}
	if result.Personality.Profile != "concise" {
		t.Errorf("Profile = %q, want %q (from project)", result.Personality.Profile, "concise")
	}
	// Checks should be merged via maps.Copy: project overwrites global keys, both present
	if _, ok := result.Personality.Checks["humor"]; !ok {
		t.Error("humor check should be preserved from global")
	}
	if _, ok := result.Personality.Checks["verbosity"]; !ok {
		t.Error("verbosity check should be set from project")
	}
}

func TestMerge_TelemetrySettings(t *testing.T) {
	t.Parallel()

	f := false
	global := &Settings{
		Telemetry: &TelemetrySettings{BudgetUSD: 10.0, WarnAtPct: 80},
	}
	project := &Settings{
		Telemetry: &TelemetrySettings{Enabled: &f, BudgetUSD: 25.0},
	}

	result := merge(global, project)

	if result.Telemetry == nil {
		t.Fatal("Telemetry should be set")
	}
	if result.Telemetry.IsEnabled() {
		t.Error("project should override Enabled to false")
	}
	if result.Telemetry.BudgetUSD != 25.0 {
		t.Errorf("BudgetUSD = %f, want 25.0 (from project)", result.Telemetry.BudgetUSD)
	}
	if result.Telemetry.WarnAtPct != 80 {
		t.Errorf("WarnAtPct = %d, want 80 (from global)", result.Telemetry.WarnAtPct)
	}
}

func TestMerge_SafetySettings(t *testing.T) {
	t.Parallel()

	global := &Settings{
		Safety: &SafetySettings{
			NeverModify: []string{"*.env"},
			LockedKeys:  []string{"model"},
		},
	}
	project := &Settings{
		Safety: &SafetySettings{
			NeverModify: []string{"secrets/*"},
			LockedKeys:  []string{"baseURL"},
		},
	}

	result := merge(global, project)

	if result.Safety == nil {
		t.Fatal("Safety should be set")
	}
	// NeverModify should be unioned with dedup
	if !containsAll(result.Safety.NeverModify, "*.env", "secrets/*") {
		t.Errorf("NeverModify = %v, want both *.env and secrets/*", result.Safety.NeverModify)
	}
	if !containsAll(result.Safety.LockedKeys, "model", "baseURL") {
		t.Errorf("LockedKeys = %v, want both model and baseURL", result.Safety.LockedKeys)
	}
}

func TestLoadFile_NewSettingsFields(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	data := `{
		"intent": {
			"enabled": false,
			"heuristicThreshold": 0.85,
			"autoPlanFileCount": 8
		},
		"prompts": {
			"activeVersion": "v2.0.0",
			"overridesDir": "/tmp/prompts",
			"maxSystemPromptTokens": 8192
		},
		"personality": {
			"profile": "concise",
			"checks": {
				"humor": {"enabled": true, "level": "minimal"}
			}
		},
		"telemetry": {
			"enabled": true,
			"budgetUsd": 50.0,
			"warnAtPct": 90
		},
		"safety": {
			"neverModify": ["*.env", ".git/*"],
			"lockedKeys": ["model", "baseURL"]
		}
	}`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}

	s, err := loadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	// Intent
	if s.Intent == nil {
		t.Fatal("Intent should be set")
	}
	if s.Intent.IsEnabled() {
		t.Error("Intent should be disabled")
	}
	if s.Intent.HeuristicThreshold != 0.85 {
		t.Errorf("HeuristicThreshold = %f, want 0.85", s.Intent.HeuristicThreshold)
	}
	if s.Intent.AutoPlanFileCount != 8 {
		t.Errorf("AutoPlanFileCount = %d, want 8", s.Intent.AutoPlanFileCount)
	}

	// Prompts
	if s.Prompts == nil {
		t.Fatal("Prompts should be set")
	}
	if s.Prompts.ActiveVersion != "v2.0.0" {
		t.Errorf("ActiveVersion = %q, want %q", s.Prompts.ActiveVersion, "v2.0.0")
	}
	if s.Prompts.MaxSystemPromptTokens != 8192 {
		t.Errorf("MaxSystemPromptTokens = %d, want 8192", s.Prompts.MaxSystemPromptTokens)
	}

	// Personality
	if s.Personality == nil {
		t.Fatal("Personality should be set")
	}
	if s.Personality.Profile != "concise" {
		t.Errorf("Profile = %q, want %q", s.Personality.Profile, "concise")
	}
	if check, ok := s.Personality.Checks["humor"]; !ok {
		t.Error("humor check should exist")
	} else if !check.IsEnabled() {
		t.Error("humor check should be enabled")
	}

	// Telemetry
	if s.Telemetry == nil {
		t.Fatal("Telemetry should be set")
	}
	if !s.Telemetry.IsEnabled() {
		t.Error("Telemetry should be enabled")
	}
	if s.Telemetry.BudgetUSD != 50.0 {
		t.Errorf("BudgetUSD = %f, want 50.0", s.Telemetry.BudgetUSD)
	}

	// Safety
	if s.Safety == nil {
		t.Fatal("Safety should be set")
	}
	if len(s.Safety.NeverModify) != 2 {
		t.Errorf("NeverModify length = %d, want 2", len(s.Safety.NeverModify))
	}
	if len(s.Safety.LockedKeys) != 2 {
		t.Errorf("LockedKeys length = %d, want 2", len(s.Safety.LockedKeys))
	}
}

// --- WorktreeSettings tests ---

func TestWorktreeSettings_IsEnabled_Default(t *testing.T) {
	t.Parallel()
	var ws *WorktreeSettings
	if !ws.IsEnabled() {
		t.Error("nil WorktreeSettings should default to enabled")
	}
}

func TestWorktreeSettings_IsEnabled_NilEnabled(t *testing.T) {
	t.Parallel()
	ws := &WorktreeSettings{}
	if !ws.IsEnabled() {
		t.Error("WorktreeSettings{} should default to enabled")
	}
}

func TestWorktreeSettings_IsEnabled_True(t *testing.T) {
	t.Parallel()
	ws := &WorktreeSettings{Enabled: new(true)}
	if !ws.IsEnabled() {
		t.Error("WorktreeSettings{Enabled: true} should be enabled")
	}
}

func TestWorktreeSettings_IsEnabled_False(t *testing.T) {
	t.Parallel()
	ws := &WorktreeSettings{Enabled: new(false)}
	if ws.IsEnabled() {
		t.Error("WorktreeSettings{Enabled: false} should be disabled")
	}
}

func TestMerge_Worktree(t *testing.T) {
	t.Parallel()

	global := &Settings{Worktree: &WorktreeSettings{Enabled: new(true)}}
	project := &Settings{Worktree: &WorktreeSettings{Enabled: new(false)}}

	result := merge(global, project)
	if result.Worktree == nil {
		t.Fatal("Worktree should not be nil after merge")
	}
	if result.Worktree.IsEnabled() {
		t.Error("project should override global: want disabled")
	}
}

func TestMerge_Worktree_NilProject(t *testing.T) {
	t.Parallel()

	global := &Settings{Worktree: &WorktreeSettings{Enabled: new(true)}}
	project := &Settings{}

	result := merge(global, project)
	if result.Worktree == nil || !result.Worktree.IsEnabled() {
		t.Error("nil project worktree should preserve global")
	}
}

// boolPtr is a test helper that returns a pointer to a bool value.
//
//go:fix inline
func boolPtr(b bool) *bool {
	return new(b)
}

// --- MinionSettings tests ---

func TestMinionSettings_Defaults(t *testing.T) {
	t.Parallel()

	var ms *MinionSettings // nil

	if ms.IsEnabled() {
		t.Error("nil MinionSettings should be disabled by default")
	}
	if ms.EffectiveModel() != "claude-haiku-4-5-20251001" {
		t.Errorf("EffectiveModel = %q, want %q", ms.EffectiveModel(), "claude-haiku-4-5-20251001")
	}
	if ms.EffectiveMode() != "singular" {
		t.Errorf("EffectiveMode = %q, want %q", ms.EffectiveMode(), "singular")
	}
}

func TestMinionSettings_CustomValues(t *testing.T) {
	t.Parallel()

	tr := true
	ms := &MinionSettings{
		Enabled: &tr,
		Model:   "llama3.2:8b",
		Mode:    "plural",
	}

	if !ms.IsEnabled() {
		t.Error("should be enabled")
	}
	if ms.EffectiveModel() != "llama3.2:8b" {
		t.Errorf("EffectiveModel = %q, want %q", ms.EffectiveModel(), "llama3.2:8b")
	}
	if ms.EffectiveMode() != "plural" {
		t.Errorf("EffectiveMode = %q, want %q", ms.EffectiveMode(), "plural")
	}
}

func TestMerge_MinionSettings(t *testing.T) {
	t.Parallel()

	tr := true
	global := &Settings{
		Minion: &MinionSettings{
			Enabled: &tr,
			Model:   "claude-haiku-4-5-20251001",
		},
	}
	project := &Settings{
		Minion: &MinionSettings{
			Mode: "plural",
		},
	}

	result := merge(global, project)

	if result.Minion == nil {
		t.Fatal("Minion should be set")
	}
	if !result.Minion.IsEnabled() {
		t.Error("Enabled should be preserved from global")
	}
	if result.Minion.Model != "claude-haiku-4-5-20251001" {
		t.Errorf("Model = %q, want %q (from global)", result.Minion.Model, "claude-haiku-4-5-20251001")
	}
	if result.Minion.Mode != "plural" {
		t.Errorf("Mode = %q, want %q (from project)", result.Minion.Mode, "plural")
	}
}

func TestMerge_MinionSettings_NoAliasing(t *testing.T) {
	t.Parallel()

	enabled := true
	global := &Settings{
		Minion: &MinionSettings{
			Enabled: &enabled,
			Model:   "llama3:8b",
		},
	}
	project := &Settings{
		Minion: &MinionSettings{
			Model: "mistral:7b",
		},
	}

	result := merge(global, project)

	if result.Minion.Model != "mistral:7b" {
		t.Errorf("expected mistral:7b, got %s", result.Minion.Model)
	}
	// global must not be mutated
	if global.Minion.Model != "llama3:8b" {
		t.Errorf("global.Minion was mutated: Model = %s", global.Minion.Model)
	}
}

func TestMerge_TelemetrySettings_NoAliasing(t *testing.T) {
	t.Parallel()

	enabled := true
	global := &Settings{
		Telemetry: &TelemetrySettings{
			Enabled:   &enabled,
			BudgetUSD: 10.0,
		},
	}
	project := &Settings{
		Telemetry: &TelemetrySettings{
			BudgetUSD: 50.0,
		},
	}

	result := merge(global, project)

	if result.Telemetry.BudgetUSD != 50.0 {
		t.Errorf("expected 50.0, got %f", result.Telemetry.BudgetUSD)
	}
	// global must not be mutated
	if global.Telemetry.BudgetUSD != 10.0 {
		t.Errorf("global.Telemetry was mutated: BudgetUSD = %f", global.Telemetry.BudgetUSD)
	}
}

func TestMerge_SafetySettings_NoAliasing(t *testing.T) {
	t.Parallel()

	global := &Settings{
		Safety: &SafetySettings{
			NeverModify: []string{"/etc/passwd"},
		},
	}
	project := &Settings{
		Safety: &SafetySettings{
			NeverModify: []string{"/etc/shadow"},
		},
	}

	result := merge(global, project)

	if len(result.Safety.NeverModify) != 2 {
		t.Errorf("expected 2 NeverModify entries, got %d", len(result.Safety.NeverModify))
	}
	// global must not be mutated
	if len(global.Safety.NeverModify) != 1 || global.Safety.NeverModify[0] != "/etc/passwd" {
		t.Errorf("global.Safety was mutated: NeverModify = %v", global.Safety.NeverModify)
	}
}

func TestMerge_WorktreeSettings_NoAliasing(t *testing.T) {
	t.Parallel()

	enabled := true
	global := &Settings{
		Worktree: &WorktreeSettings{
			Enabled: &enabled,
		},
	}
	disabled := false
	project := &Settings{
		Worktree: &WorktreeSettings{
			Enabled: &disabled,
		},
	}

	result := merge(global, project)

	if *result.Worktree.Enabled != false {
		t.Error("expected Worktree.Enabled = false from project")
	}
	// global must not be mutated
	if *global.Worktree.Enabled != true {
		t.Error("global.Worktree was mutated: Enabled should still be true")
	}
}

func TestLoadFile_MinionSettings(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	data := `{"minion":{"enabled":true,"model":"claude-haiku-4-5-20251001","mode":"singular"}}`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}

	s, err := loadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if s.Minion == nil {
		t.Fatal("Minion should be set")
	}
	if !s.Minion.IsEnabled() {
		t.Error("should be enabled")
	}
	if s.Minion.Model != "claude-haiku-4-5-20251001" {
		t.Errorf("Model = %q, want %q", s.Minion.Model, "claude-haiku-4-5-20251001")
	}
	if s.Minion.Mode != "singular" {
		t.Errorf("Mode = %q, want %q", s.Minion.Mode, "singular")
	}
}

func TestGatewaySettings_ResolveBaseURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		gw   *GatewaySettings
		api  string
		want string
	}{
		{"nil gateway", nil, "anthropic", ""},
		{"empty URL", &GatewaySettings{}, "anthropic", ""},
		{"default anthropic", &GatewaySettings{URL: "http://localhost:8080"}, "anthropic", "http://localhost:8080/anthropic"},
		{"default openai", &GatewaySettings{URL: "http://localhost:8080"}, "openai", "http://localhost:8080/openai"},
		{"default google", &GatewaySettings{URL: "http://localhost:8080"}, "google", "http://localhost:8080/gemini/v1beta"},
		{"default vertex", &GatewaySettings{URL: "http://localhost:8080"}, "vertex", "http://localhost:8080/vertex/v1"},
		{"trailing slash stripped", &GatewaySettings{URL: "http://localhost:8080/"}, "anthropic", "http://localhost:8080/anthropic"},
		{
			"custom path override",
			&GatewaySettings{URL: "http://gw:9090", Paths: map[string]string{"anthropic": "/api/claude"}},
			"anthropic",
			"http://gw:9090/api/claude",
		},
		{"unknown api gets bare gateway URL", &GatewaySettings{URL: "http://localhost:8080"}, "bedrock", "http://localhost:8080"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.gw.ResolveBaseURL(tt.api)
			if got != tt.want {
				t.Errorf("ResolveBaseURL(%q) = %q, want %q", tt.api, got, tt.want)
			}
		})
	}
}

func TestGatewaySettings_JSON(t *testing.T) {
	t.Parallel()

	s := Settings{
		Gateway: &GatewaySettings{
			URL:   "http://localhost:8080",
			Paths: map[string]string{"anthropic": "/custom"},
		},
	}

	data, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}

	var decoded Settings
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Gateway == nil {
		t.Fatal("Gateway should not be nil")
	}
	if decoded.Gateway.URL != "http://localhost:8080" {
		t.Errorf("URL = %q", decoded.Gateway.URL)
	}
	if decoded.Gateway.Paths["anthropic"] != "/custom" {
		t.Errorf("Paths = %v", decoded.Gateway.Paths)
	}
}

func TestMerge_Gateway(t *testing.T) {
	t.Parallel()

	t.Run("project overrides global URL", func(t *testing.T) {
		global := &Settings{Gateway: &GatewaySettings{URL: "http://global:8080"}}
		project := &Settings{Gateway: &GatewaySettings{URL: "http://project:9090"}}
		merged := merge(global, project)
		if merged.Gateway == nil || merged.Gateway.URL != "http://project:9090" {
			t.Errorf("Gateway.URL = %q, want http://project:9090", merged.Gateway.URL)
		}
	})

	t.Run("project paths merge over global", func(t *testing.T) {
		global := &Settings{Gateway: &GatewaySettings{
			URL:   "http://gw:8080",
			Paths: map[string]string{"anthropic": "/anthropic", "openai": "/openai"},
		}}
		project := &Settings{Gateway: &GatewaySettings{
			Paths: map[string]string{"anthropic": "/custom-anthropic"},
		}}
		merged := merge(global, project)
		if merged.Gateway.URL != "http://gw:8080" {
			t.Errorf("URL = %q, want http://gw:8080 (preserved from global)", merged.Gateway.URL)
		}
		if merged.Gateway.Paths["anthropic"] != "/custom-anthropic" {
			t.Errorf("anthropic = %q, want /custom-anthropic", merged.Gateway.Paths["anthropic"])
		}
		if merged.Gateway.Paths["openai"] != "/openai" {
			t.Errorf("openai = %q, want /openai (preserved from global)", merged.Gateway.Paths["openai"])
		}
	})

	t.Run("nil global + project gateway", func(t *testing.T) {
		merged := merge(&Settings{}, &Settings{Gateway: &GatewaySettings{URL: "http://new:8080"}})
		if merged.Gateway == nil || merged.Gateway.URL != "http://new:8080" {
			t.Errorf("Gateway = %v, want URL http://new:8080", merged.Gateway)
		}
	})

	t.Run("global gateway + nil project preserves global", func(t *testing.T) {
		merged := merge(&Settings{Gateway: &GatewaySettings{URL: "http://global:8080"}}, &Settings{})
		if merged.Gateway == nil || merged.Gateway.URL != "http://global:8080" {
			t.Errorf("Gateway = %v, want URL http://global:8080", merged.Gateway)
		}
	})

	t.Run("merge does not mutate original", func(t *testing.T) {
		global := &Settings{Gateway: &GatewaySettings{
			URL:   "http://gw:8080",
			Paths: map[string]string{"anthropic": "/anthropic"},
		}}
		project := &Settings{Gateway: &GatewaySettings{
			Paths: map[string]string{"openai": "/openai"},
		}}
		_ = merge(global, project)
		if _, ok := global.Gateway.Paths["openai"]; ok {
			t.Error("merge mutated global Gateway.Paths")
		}
	})
}
