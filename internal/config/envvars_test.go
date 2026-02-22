// ABOUTME: Tests for environment variable expansion in config
// ABOUTME: Validates ${VAR} replacement for set, unset, and nested patterns

package config

import (
	"testing"
)

func TestExpandEnv_Set(t *testing.T) {
	t.Setenv("TEST_MODEL", "claude-sonnet")
	result := expandEnv("${TEST_MODEL}")
	if result != "claude-sonnet" {
		t.Errorf("expandEnv = %q; want %q", result, "claude-sonnet")
	}
}

func TestExpandEnv_Unset(t *testing.T) {
	result := expandEnv("${DEFINITELY_NOT_SET_12345}")
	if result != "" {
		t.Errorf("expandEnv = %q; want empty for unset var", result)
	}
}

func TestExpandEnv_Mixed(t *testing.T) {
	t.Setenv("MY_HOST", "localhost")
	result := expandEnv("https://${MY_HOST}:8080/v1")
	if result != "https://localhost:8080/v1" {
		t.Errorf("expandEnv = %q; want %q", result, "https://localhost:8080/v1")
	}
}

func TestExpandEnv_NoPattern(t *testing.T) {
	result := expandEnv("plain string")
	if result != "plain string" {
		t.Errorf("expandEnv = %q; want %q", result, "plain string")
	}
}

func TestExpandEnv_Empty(t *testing.T) {
	result := expandEnv("")
	if result != "" {
		t.Errorf("expandEnv = %q; want empty", result)
	}
}

func TestResolveEnvVars_SettingsFields(t *testing.T) {
	t.Setenv("TEST_BASE_URL", "https://api.example.com")
	t.Setenv("TEST_CMD", "echo hello")

	s := &Settings{
		BaseURL: "${TEST_BASE_URL}",
		StatusLine: &StatusLineConfig{
			Command: "${TEST_CMD}",
		},
		Hooks: map[string][]HookDef{
			"PreToolUse": {
				{Command: "${TEST_CMD}", Matcher: "bash"},
			},
		},
		Env: map[string]string{
			"key": "${TEST_BASE_URL}/path",
		},
	}

	ResolveEnvVars(s)

	if s.BaseURL != "https://api.example.com" {
		t.Errorf("BaseURL = %q; want %q", s.BaseURL, "https://api.example.com")
	}
	if s.StatusLine.Command != "echo hello" {
		t.Errorf("StatusLine.Command = %q; want %q", s.StatusLine.Command, "echo hello")
	}
	if s.Hooks["PreToolUse"][0].Command != "echo hello" {
		t.Errorf("Hook Command = %q; want %q", s.Hooks["PreToolUse"][0].Command, "echo hello")
	}
	if s.Env["key"] != "https://api.example.com/path" {
		t.Errorf("Env[key] = %q; want %q", s.Env["key"], "https://api.example.com/path")
	}
}
