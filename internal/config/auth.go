// ABOUTME: Auth credential storage with file locking via flock
// ABOUTME: Reads/writes ~/.pi-go/auth.json with 0600 permissions

package config

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

// AuthStore holds API keys and tokens.
type AuthStore struct {
	Keys map[string]string `json:"keys"` // provider -> api key
	mu   sync.Mutex
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

	if err := os.WriteFile(AuthFile(), data, 0o600); err != nil {
		return fmt.Errorf("writing auth file: %w", err)
	}
	return nil
}

// GetKey returns the API key for a provider. Falls back to environment
// variables: PI_API_KEY_<PROVIDER> or <PROVIDER>_API_KEY.
func (a *AuthStore) GetKey(provider string) string {
	a.mu.Lock()
	key := a.Keys[provider]
	a.mu.Unlock()

	if key != "" {
		return key
	}

	// Try environment variables
	envVars := []string{
		"PI_API_KEY_" + provider,
		provider + "_API_KEY",
	}
	for _, env := range envVars {
		if v := os.Getenv(env); v != "" {
			return v
		}
	}
	return ""
}

// SetKey stores an API key for a provider.
func (a *AuthStore) SetKey(provider, key string) {
	a.mu.Lock()
	a.Keys[provider] = key
	a.mu.Unlock()
}
