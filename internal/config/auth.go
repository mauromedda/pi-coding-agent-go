// ABOUTME: Auth credential storage with file locking via flock
// ABOUTME: Reads/writes ~/.pi-go/auth.json with 0600 permissions

package config

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"

	"golang.org/x/sync/singleflight"
)

// AuthStore holds API keys and tokens.
type AuthStore struct {
	Keys       map[string]string `json:"keys"` // provider -> api key
	mu         sync.Mutex
	runtimeKey string            // CLI --api-key override; not persisted
	cmdCache   map[string]string // per-process cache for !command resolutions
	cmdGroup   singleflight.Group // deduplicates concurrent resolveCommandKey calls
}

// LoadAuth reads the auth file, or returns an empty store if it doesn't exist.
func LoadAuth() (*AuthStore, error) {
	store := &AuthStore{Keys: make(map[string]string)}
	data, err := os.ReadFile(AuthFile())
	if os.IsNotExist(err) {
		return store, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading auth file: %w", err)
	}
	if err := json.Unmarshal(data, store); err != nil {
		return nil, fmt.Errorf("parsing auth file: %w", err)
	}
	return store, nil
}

// Save writes the auth store to disk with restricted permissions.
// Uses atomic write (temp file + rename) to prevent partial writes on crash.
func (a *AuthStore) Save() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if err := EnsureDir(GlobalDir()); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}

	data, err := json.MarshalIndent(a, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling auth: %w", err)
	}

	target := AuthFile()
	tmp := target + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("writing temp auth file: %w", err)
	}
	if err := os.Rename(tmp, target); err != nil {
		os.Remove(tmp) // Best-effort cleanup
		return fmt.Errorf("renaming auth file: %w", err)
	}
	return nil
}

// SetRuntimeKey sets a CLI-provided API key that overrides all other sources.
func (a *AuthStore) SetRuntimeKey(key string) {
	a.mu.Lock()
	a.runtimeKey = key
	a.mu.Unlock()
}

// GetKey returns the API key for a provider using the priority chain:
// runtimeKey > stored key (with !command resolution) > env var > empty.
func (a *AuthStore) GetKey(provider string) string {
	a.mu.Lock()
	runtime := a.runtimeKey
	key := a.Keys[provider]
	a.mu.Unlock()

	// Priority 1: CLI runtime override
	if runtime != "" {
		return runtime
	}

	// Priority 2: Stored key (may be a !command)
	if key != "" {
		if strings.HasPrefix(key, "!") {
			resolved, err := a.resolveCommandKey(key[1:])
			if err == nil && resolved != "" {
				return resolved
			}
			// Fall through to env vars on command error
		} else {
			return key
		}
	}

	// Priority 3: Environment variables
	upper := strings.ToUpper(provider)
	envVars := []string{
		"PI_API_KEY_" + upper,
		upper + "_API_KEY",
	}
	for _, env := range envVars {
		if v := os.Getenv(env); v != "" {
			return v
		}
	}
	return ""
}

// resolveCommandKey executes a shell command and returns its trimmed output.
// Results are cached per-process. Concurrent calls for the same command are
// deduplicated via singleflight to avoid the TOCTOU race on cache lookup.
func (a *AuthStore) resolveCommandKey(cmd string) (string, error) {
	a.mu.Lock()
	if a.cmdCache != nil {
		if cached, ok := a.cmdCache[cmd]; ok {
			a.mu.Unlock()
			return cached, nil
		}
	}
	a.mu.Unlock()

	v, err, _ := a.cmdGroup.Do(cmd, func() (any, error) {
		// Double-check cache inside singleflight to handle the case where a
		// previous flight already populated the cache.
		a.mu.Lock()
		if a.cmdCache != nil {
			if cached, ok := a.cmdCache[cmd]; ok {
				a.mu.Unlock()
				return cached, nil
			}
		}
		a.mu.Unlock()

		out, err := exec.Command("/bin/sh", "-c", cmd).Output()
		if err != nil {
			return "", fmt.Errorf("executing key command %q: %w", cmd, err)
		}

		result := strings.TrimSpace(string(out))

		a.mu.Lock()
		if a.cmdCache == nil {
			a.cmdCache = make(map[string]string)
		}
		a.cmdCache[cmd] = result
		a.mu.Unlock()

		return result, nil
	})
	if err != nil {
		return "", err
	}

	return v.(string), nil
}

// SetKey stores an API key for a provider.
func (a *AuthStore) SetKey(provider, key string) {
	a.mu.Lock()
	a.Keys[provider] = key
	a.mu.Unlock()
}
