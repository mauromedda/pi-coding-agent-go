// ABOUTME: Tests for ~/.pi/agent/ config compatibility layer
// ABOUTME: Covers PiAgentDir, LoadPiCompat, convertPiApiType, MergePiAuth

package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/mauromedda/pi-coding-agent-go/pkg/ai"
)

func TestPiAgentDir_NonExistent(t *testing.T) {
	t.Parallel()

	dir := PiAgentDirFrom("/nonexistent/path/surely")
	if dir != "" {
		t.Errorf("expected empty string for nonexistent home, got %q", dir)
	}
}

func TestPiAgentDir_Exists(t *testing.T) {
	t.Parallel()

	home := t.TempDir()
	agentDir := filepath.Join(home, ".pi", "agent")
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		t.Fatal(err)
	}

	got := PiAgentDirFrom(home)
	if got != agentDir {
		t.Errorf("expected %q, got %q", agentDir, got)
	}
}

func TestConvertPiApiType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  ai.Api
	}{
		{"openai-completions", ai.ApiOpenAI},
		{"openai", ai.ApiOpenAI},
		{"anthropic", ai.ApiAnthropic},
		{"google", ai.ApiGoogle},
		{"vertex", ai.ApiVertex},
		{"unknown-api", ai.ApiOpenAI}, // fallback
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			got := convertPiApiType(tt.input)
			if got != tt.want {
				t.Errorf("convertPiApiType(%q) = %q; want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestLoadPiCompat_ValidVLLM(t *testing.T) {
	t.Parallel()

	piDir := setupPiAgentDir(t,
		`{
			"defaultProvider": "vllm",
			"defaultModel": "Qwen/Qwen3-Coder-Next-FP8",
			"defaultThinkingLevel": "off"
		}`,
		`{
			"providers": {
				"vllm": {
					"baseUrl": "http://spark.prjxai.xyz:8000/v1",
					"api": "openai-completions",
					"apiKey": "vllm",
					"models": [
						{
							"id": "Qwen/Qwen3-Coder-Next-FP8",
							"name": "Qwen3 Coder Next",
							"contextWindow": 128000,
							"maxTokens": 8192
						}
					]
				}
			}
		}`,
	)

	settings, apiKeys, err := LoadPiCompat(piDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	wantModel := "vllm:Qwen/Qwen3-Coder-Next-FP8"
	if settings.Model != wantModel {
		t.Errorf("Model = %q; want %q", settings.Model, wantModel)
	}

	wantBaseURL := "http://spark.prjxai.xyz:8000/v1"
	if settings.BaseURL != wantBaseURL {
		t.Errorf("BaseURL = %q; want %q", settings.BaseURL, wantBaseURL)
	}

	if settings.Thinking {
		t.Error("expected Thinking=false when defaultThinkingLevel=off")
	}

	if apiKeys["vllm"] != "vllm" {
		t.Errorf("expected apiKey 'vllm' for provider vllm, got %q", apiKeys["vllm"])
	}
}

func TestLoadPiCompat_ThinkingOn(t *testing.T) {
	t.Parallel()

	piDir := setupPiAgentDir(t,
		`{
			"defaultProvider": "anthropic",
			"defaultModel": "claude-sonnet-4-6",
			"defaultThinkingLevel": "medium"
		}`,
		`{
			"providers": {
				"anthropic": {
					"baseUrl": "",
					"api": "anthropic",
					"apiKey": "sk-ant-xxx",
					"models": []
				}
			}
		}`,
	)

	settings, _, err := LoadPiCompat(piDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !settings.Thinking {
		t.Error("expected Thinking=true when defaultThinkingLevel != off")
	}
}

func TestLoadPiCompat_MissingSettingsFile(t *testing.T) {
	t.Parallel()

	piDir := t.TempDir()
	// No settings.json, but models.json exists
	modelsData := `{"providers": {}}`
	if err := os.WriteFile(filepath.Join(piDir, "models.json"), []byte(modelsData), 0o644); err != nil {
		t.Fatal(err)
	}

	settings, _, err := LoadPiCompat(piDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should return zero-value settings gracefully
	if settings.Model != "" {
		t.Errorf("expected empty model, got %q", settings.Model)
	}
}

func TestLoadPiCompat_MissingModelsFile(t *testing.T) {
	t.Parallel()

	piDir := t.TempDir()
	// Only settings.json
	settingsData := `{"defaultProvider": "vllm", "defaultModel": "test-model"}`
	if err := os.WriteFile(filepath.Join(piDir, "settings.json"), []byte(settingsData), 0o644); err != nil {
		t.Fatal(err)
	}

	settings, apiKeys, err := LoadPiCompat(piDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Model should still resolve from settings.json alone
	if settings.Model != "vllm:test-model" {
		t.Errorf("expected 'vllm:test-model', got %q", settings.Model)
	}
	if len(apiKeys) != 0 {
		t.Errorf("expected empty apiKeys, got %v", apiKeys)
	}
}

func TestLoadPiCompat_MaxTokensFromModel(t *testing.T) {
	t.Parallel()

	piDir := setupPiAgentDir(t,
		`{
			"defaultProvider": "vllm",
			"defaultModel": "test-model"
		}`,
		`{
			"providers": {
				"vllm": {
					"baseUrl": "http://localhost:8000/v1",
					"api": "openai-completions",
					"apiKey": "tok",
					"models": [
						{
							"id": "test-model",
							"maxTokens": 16384
						}
					]
				}
			}
		}`,
	)

	settings, _, err := LoadPiCompat(piDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if settings.MaxTokens != 16384 {
		t.Errorf("MaxTokens = %d; want 16384", settings.MaxTokens)
	}
}

func TestMergePiAuth_InjectsKeys(t *testing.T) {
	t.Parallel()

	piDir := setupPiAgentDir(t,
		`{}`,
		`{
			"providers": {
				"vllm": {
					"apiKey": "vllm-token",
					"models": []
				},
				"anthropic": {
					"apiKey": "sk-ant-secret",
					"models": []
				}
			}
		}`,
	)

	store := &AuthStore{Keys: make(map[string]string)}
	MergePiAuth(store, piDir)

	if got := store.Keys["vllm"]; got != "vllm-token" {
		t.Errorf("expected vllm key 'vllm-token', got %q", got)
	}
	if got := store.Keys["anthropic"]; got != "sk-ant-secret" {
		t.Errorf("expected anthropic key 'sk-ant-secret', got %q", got)
	}
}

func TestMergePiAuth_DoesNotOverrideExisting(t *testing.T) {
	t.Parallel()

	piDir := setupPiAgentDir(t,
		`{}`,
		`{
			"providers": {
				"anthropic": {
					"apiKey": "pi-key",
					"models": []
				}
			}
		}`,
	)

	store := &AuthStore{Keys: map[string]string{"anthropic": "existing-key"}}
	MergePiAuth(store, piDir)

	if got := store.Keys["anthropic"]; got != "existing-key" {
		t.Errorf("expected existing key preserved, got %q", got)
	}
}

func TestMergePiAuth_EmptyDir(t *testing.T) {
	t.Parallel()

	store := &AuthStore{Keys: make(map[string]string)}
	MergePiAuth(store, "") // empty dir should be a no-op

	if len(store.Keys) != 0 {
		t.Errorf("expected no keys after empty dir merge, got %v", store.Keys)
	}
}

// setupPiAgentDir creates a temp directory mimicking ~/.pi/agent/ with the given JSON files.
func setupPiAgentDir(t *testing.T, settingsJSON, modelsJSON string) string {
	t.Helper()
	dir := t.TempDir()

	if settingsJSON != "" {
		// Validate JSON
		var raw json.RawMessage
		if err := json.Unmarshal([]byte(settingsJSON), &raw); err != nil {
			t.Fatalf("invalid settings JSON: %v", err)
		}
		if err := os.WriteFile(filepath.Join(dir, "settings.json"), []byte(settingsJSON), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	if modelsJSON != "" {
		var raw json.RawMessage
		if err := json.Unmarshal([]byte(modelsJSON), &raw); err != nil {
			t.Fatalf("invalid models JSON: %v", err)
		}
		if err := os.WriteFile(filepath.Join(dir, "models.json"), []byte(modelsJSON), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	return dir
}
