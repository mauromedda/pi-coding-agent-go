// ABOUTME: Tests for five-level settings precedence and new settings fields
// ABOUTME: Verifies user -> project -> local -> CLI -> managed override chain

package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestLoadAll_Empty(t *testing.T) {
	t.Parallel()

	project := t.TempDir()
	s, err := LoadAllWithHome(project, t.TempDir(), nil)
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	if s == nil {
		t.Fatal("expected non-nil settings")
	}
}

func TestLoadAll_UserSettings(t *testing.T) {
	t.Parallel()

	project := t.TempDir()
	home := t.TempDir()

	mkDir(t, filepath.Join(home, ".pi-go"))
	writeJSON(t, filepath.Join(home, ".pi-go", "settings.json"), `{"model":"user-model"}`)

	s, err := LoadAllWithHome(project, home, nil)
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	if s.Model != "user-model" {
		t.Errorf("expected user-model, got %q", s.Model)
	}
}

func TestLoadAll_ProjectOverridesUser(t *testing.T) {
	t.Parallel()

	project := t.TempDir()
	home := t.TempDir()

	mkDir(t, filepath.Join(home, ".pi-go"))
	writeJSON(t, filepath.Join(home, ".pi-go", "settings.json"), `{"model":"user-model","temperature":0.5}`)

	mkDir(t, filepath.Join(project, ".pi-go"))
	writeJSON(t, filepath.Join(project, ".pi-go", "settings.json"), `{"model":"project-model"}`)

	s, err := LoadAllWithHome(project, home, nil)
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	if s.Model != "project-model" {
		t.Errorf("expected project-model, got %q", s.Model)
	}
	if s.Temperature != 0.5 {
		t.Errorf("expected temperature 0.5 from user, got %f", s.Temperature)
	}
}

func TestLoadAll_LocalOverridesProject(t *testing.T) {
	t.Parallel()

	project := t.TempDir()
	home := t.TempDir()

	mkDir(t, filepath.Join(project, ".pi-go"))
	writeJSON(t, filepath.Join(project, ".pi-go", "settings.json"), `{"model":"project-model"}`)
	writeJSON(t, filepath.Join(project, ".pi-go", "settings.local.json"), `{"model":"local-model"}`)

	s, err := LoadAllWithHome(project, home, nil)
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	if s.Model != "local-model" {
		t.Errorf("expected local-model, got %q", s.Model)
	}
}

func TestLoadAll_CLIOverridesLocal(t *testing.T) {
	t.Parallel()

	project := t.TempDir()
	home := t.TempDir()

	mkDir(t, filepath.Join(project, ".pi-go"))
	writeJSON(t, filepath.Join(project, ".pi-go", "settings.local.json"), `{"model":"local-model"}`)

	cli := &Settings{Model: "cli-model"}
	s, err := LoadAllWithHome(project, home, cli)
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	if s.Model != "cli-model" {
		t.Errorf("expected cli-model, got %q", s.Model)
	}
}

func TestLoadAll_NewFields(t *testing.T) {
	t.Parallel()

	project := t.TempDir()
	home := t.TempDir()

	mkDir(t, filepath.Join(project, ".pi-go"))
	writeJSON(t, filepath.Join(project, ".pi-go", "settings.json"), `{
		"allow": ["Bash(npm run *)"],
		"deny": ["Bash(rm -rf *)"],
		"ask": ["Write"],
		"sandbox": {"excludedCommands": ["curl"], "allowedDomains": ["example.com"]}
	}`)

	s, err := LoadAllWithHome(project, home, nil)
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	if len(s.Allow) != 1 || s.Allow[0] != "Bash(npm run *)" {
		t.Errorf("unexpected Allow: %v", s.Allow)
	}
	if len(s.Deny) != 1 || s.Deny[0] != "Bash(rm -rf *)" {
		t.Errorf("unexpected Deny: %v", s.Deny)
	}
	if len(s.Ask) != 1 || s.Ask[0] != "Write" {
		t.Errorf("unexpected Ask: %v", s.Ask)
	}
	if len(s.Sandbox.ExcludedCommands) != 1 {
		t.Errorf("unexpected Sandbox.ExcludedCommands: %v", s.Sandbox.ExcludedCommands)
	}
}

func TestLoadAll_BackwardCompat(t *testing.T) {
	t.Parallel()

	project := t.TempDir()
	home := t.TempDir()

	// Old-style config.json should still work
	mkDir(t, filepath.Join(home, ".pi-go"))
	writeJSON(t, filepath.Join(home, ".pi-go", "config.json"), `{"model":"old-style"}`)

	s, err := LoadAllWithHome(project, home, nil)
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	if s.Model != "old-style" {
		t.Errorf("expected old-style model from config.json, got %q", s.Model)
	}
}

func TestLoadAll_SettingsOverridesConfig(t *testing.T) {
	t.Parallel()

	project := t.TempDir()
	home := t.TempDir()

	// Both config.json and settings.json exist; settings.json wins
	mkDir(t, filepath.Join(home, ".pi-go"))
	writeJSON(t, filepath.Join(home, ".pi-go", "config.json"), `{"model":"old"}`)
	writeJSON(t, filepath.Join(home, ".pi-go", "settings.json"), `{"model":"new"}`)

	s, err := LoadAllWithHome(project, home, nil)
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	if s.Model != "new" {
		t.Errorf("settings.json should override config.json, got %q", s.Model)
	}
}

func TestUserSettingsFile(t *testing.T) {
	path := UserSettingsFile()
	if path == "" {
		t.Error("UserSettingsFile returned empty")
	}
}

func TestLocalSettingsFile(t *testing.T) {
	path := LocalSettingsFile("/tmp/project")
	if path == "" {
		t.Error("LocalSettingsFile returned empty")
	}
}

func TestManagedSettingsFile(t *testing.T) {
	path := ManagedSettingsFile()
	if path == "" {
		t.Error("ManagedSettingsFile returned empty")
	}
	switch runtime.GOOS {
	case "darwin":
		if path == "" {
			t.Error("expected non-empty on darwin")
		}
	case "linux":
		if path == "" {
			t.Error("expected non-empty on linux")
		}
	}
}

func TestHookDef(t *testing.T) {
	t.Parallel()

	project := t.TempDir()
	home := t.TempDir()

	mkDir(t, filepath.Join(project, ".pi-go"))
	writeJSON(t, filepath.Join(project, ".pi-go", "settings.json"), `{
		"hooks": {
			"PreToolUse": [{"matcher": "Edit|Write", "type": "command", "command": "echo lint"}]
		}
	}`)

	s, err := LoadAllWithHome(project, home, nil)
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	hooks := s.Hooks["PreToolUse"]
	if len(hooks) != 1 {
		t.Fatalf("expected 1 PreToolUse hook, got %d", len(hooks))
	}
	if hooks[0].Command != "echo lint" {
		t.Errorf("unexpected hook command: %q", hooks[0].Command)
	}
}

// Helpers

func mkDir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
}

func writeJSON(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
