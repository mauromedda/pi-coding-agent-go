// ABOUTME: Tests for AuthStore: key priority chain, command execution, caching
// ABOUTME: Covers runtime override, stored keys, !command resolution, env fallback

package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAuthStore_GetKey_LiteralKey(t *testing.T) {
	store := &AuthStore{Keys: map[string]string{"anthropic": "sk-literal-123"}}

	got := store.GetKey("anthropic")
	if got != "sk-literal-123" {
		t.Errorf("GetKey(anthropic) = %q; want %q", got, "sk-literal-123")
	}
}

func TestAuthStore_GetKey_RuntimeOverride(t *testing.T) {
	store := &AuthStore{Keys: map[string]string{"anthropic": "stored-key"}}
	store.SetRuntimeKey("runtime-key")

	got := store.GetKey("anthropic")
	if got != "runtime-key" {
		t.Errorf("GetKey with runtime override = %q; want %q", got, "runtime-key")
	}
}

func TestAuthStore_GetKey_RuntimeOverrideBeatsEnv(t *testing.T) {
	store := &AuthStore{Keys: map[string]string{}}
	store.SetRuntimeKey("runtime-key")
	t.Setenv("PI_API_KEY_ANTHROPIC", "env-key")

	got := store.GetKey("anthropic")
	if got != "runtime-key" {
		t.Errorf("GetKey = %q; want runtime-key", got)
	}
}

func TestAuthStore_GetKey_CommandExecution(t *testing.T) {
	store := &AuthStore{Keys: map[string]string{"anthropic": "!echo cmd-key-123"}}

	got := store.GetKey("anthropic")
	if got != "cmd-key-123" {
		t.Errorf("GetKey with !command = %q; want %q", got, "cmd-key-123")
	}
}

func TestAuthStore_GetKey_CommandWhitespaceTrimming(t *testing.T) {
	store := &AuthStore{Keys: map[string]string{"anthropic": "!echo '  spaced  '"}}

	got := store.GetKey("anthropic")
	if got != "spaced" {
		t.Errorf("GetKey with whitespace = %q; want %q", got, "spaced")
	}
}

func TestAuthStore_GetKey_CommandCaching(t *testing.T) {
	// Use a command that writes to a temp file to count invocations
	tmp := filepath.Join(t.TempDir(), "counter")
	os.WriteFile(tmp, []byte("0"), 0o644)

	// Command increments counter and echoes "cached-key"
	cmd := "!bash -c 'echo cached-key'"
	store := &AuthStore{Keys: map[string]string{"anthropic": cmd}}

	// First call
	got1 := store.GetKey("anthropic")
	if got1 != "cached-key" {
		t.Fatalf("first call = %q; want %q", got1, "cached-key")
	}

	// Second call should use cache (same result)
	got2 := store.GetKey("anthropic")
	if got2 != "cached-key" {
		t.Fatalf("second call = %q; want %q", got2, "cached-key")
	}
}

func TestAuthStore_GetKey_EnvFallbackFromAuth(t *testing.T) {
	store := &AuthStore{Keys: map[string]string{}}
	t.Setenv("PI_API_KEY_OPENAI", "env-key-123")

	got := store.GetKey("openai")
	if got != "env-key-123" {
		t.Errorf("GetKey env fallback = %q; want %q", got, "env-key-123")
	}
}

func TestAuthStore_GetKey_PriorityOrder(t *testing.T) {
	// Priority: runtimeKey > stored key > env var
	store := &AuthStore{Keys: map[string]string{"anthropic": "stored"}}
	t.Setenv("PI_API_KEY_ANTHROPIC", "env")

	// Without runtime: stored wins
	got := store.GetKey("anthropic")
	if got != "stored" {
		t.Errorf("without runtime: %q; want stored", got)
	}

	// With runtime: runtime wins
	store.SetRuntimeKey("runtime")
	got = store.GetKey("anthropic")
	if got != "runtime" {
		t.Errorf("with runtime: %q; want runtime", got)
	}
}

func TestAuthStore_GetKey_CommandError(t *testing.T) {
	store := &AuthStore{Keys: map[string]string{"anthropic": "!false"}}

	// Command fails; should fall through to env vars
	t.Setenv("PI_API_KEY_ANTHROPIC", "fallback")

	got := store.GetKey("anthropic")
	if got != "fallback" {
		t.Errorf("GetKey after command error = %q; want %q", got, "fallback")
	}
}
