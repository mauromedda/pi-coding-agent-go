// ABOUTME: Tests for config loading, merging, and auth storage
// ABOUTME: Uses temp directories for isolated file-based tests

package config

import (
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
