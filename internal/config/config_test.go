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
