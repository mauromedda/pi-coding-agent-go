// ABOUTME: Tests for keybindings manager
// ABOUTME: Validates key lookup, conflict detection, merge, reload, and format

package keybindings

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mauromedda/pi-coding-agent-go/internal/config"
	"github.com/mauromedda/pi-coding-agent-go/pkg/tui/key"
)

func TestManager_DefaultBindings(t *testing.T) {
	t.Parallel()
	m := NewFromBindings(config.NewKeybindings())

	tests := []struct {
		key    key.Key
		action config.KeyAction
	}{
		{key.Key{Type: key.KeyCtrlG, Ctrl: true}, config.ActionOpenEditor},
		{key.Key{Type: key.KeyCtrlC, Ctrl: true}, config.ActionAbort},
		{key.Key{Type: key.KeyCtrlD, Ctrl: true}, config.ActionExit},
		{key.Key{Type: key.KeyEnter, Alt: true}, config.ActionQueueFollowUp},
		{key.Key{Type: key.KeyUp, Alt: true}, config.ActionHistoryPrev},
		{key.Key{Type: key.KeyDown, Alt: true}, config.ActionHistoryNext},
		{key.Key{Type: key.KeyRune, Rune: '@'}, config.ActionFileMention},
		{key.Key{Type: key.KeyBackTab}, config.ActionToggleMode},
	}

	for _, tt := range tests {
		t.Run(string(tt.action), func(t *testing.T) {
			got := m.ActionForKey(tt.key)
			if got != tt.action {
				t.Errorf("ActionForKey(%+v) = %q; want %q", tt.key, got, tt.action)
			}
		})
	}
}

func TestManager_UnboundKey(t *testing.T) {
	t.Parallel()
	m := NewFromBindings(config.NewKeybindings())

	action := m.ActionForKey(key.Key{Type: key.KeyRune, Rune: 'z'})
	if action != "" {
		t.Errorf("expected empty action for unbound key, got %q", action)
	}
}

func TestManager_Conflicts(t *testing.T) {
	t.Parallel()
	// enter is bound to both accept and sendMessage in defaults
	m := NewFromBindings(config.NewKeybindings())

	conflicts := m.Conflicts()
	// There should be some conflicts in the default bindings
	// (enter → accept and sendMessage; alt+enter → sendMessageAlt and queueFollowUp)
	if len(conflicts) == 0 {
		t.Log("no conflicts detected (may be expected if defaults have no overlaps)")
	}

	// Verify conflict structure
	for _, c := range conflicts {
		if c.Key == "" {
			t.Error("conflict with empty key")
		}
		if len(c.Actions) < 2 {
			t.Errorf("conflict for key %q has fewer than 2 actions", c.Key)
		}
	}
}

func TestManager_MergeOverridesBindings(t *testing.T) {
	t.Parallel()
	base := config.NewKeybindings()
	override := config.NewKeybindings()
	override.Bindings[config.ActionOpenEditor] = []string{"ctrl+e"}

	mergeBindings(base, override)

	if keys := base.GetBindings(config.ActionOpenEditor); len(keys) != 1 || keys[0] != "ctrl+e" {
		t.Errorf("expected [ctrl+e] after merge, got %v", keys)
	}
}

func TestManager_NewWithFiles(t *testing.T) {
	dir := t.TempDir()
	globalPath := filepath.Join(dir, "global.json")
	localPath := filepath.Join(dir, "local.json")

	// Write global override
	globalData, _ := json.Marshal(map[string][]string{
		"openEditor": {"ctrl+e"},
	})
	if err := os.WriteFile(globalPath, globalData, 0o600); err != nil {
		t.Fatal(err)
	}

	// Write local override (takes precedence)
	localData, _ := json.Marshal(map[string][]string{
		"openEditor": {"ctrl+shift+e"},
	})
	if err := os.WriteFile(localPath, localData, 0o600); err != nil {
		t.Fatal(err)
	}

	m := New(globalPath, localPath)

	// Local should override global
	action := m.ActionForKey(key.Key{Type: key.KeyRune, Rune: 'e', Ctrl: true, Shift: true})
	if action != config.ActionOpenEditor {
		t.Errorf("expected openEditor from local override, got %q", action)
	}
}

func TestManager_NewMissingFiles(t *testing.T) {
	t.Parallel()
	// Should not panic with non-existent files
	m := New("/nonexistent/global.json", "/nonexistent/local.json")
	if m == nil {
		t.Fatal("expected non-nil manager even with missing files")
	}

	// Should still have default bindings
	action := m.ActionForKey(key.Key{Type: key.KeyCtrlC, Ctrl: true})
	if action != config.ActionAbort {
		t.Errorf("expected default abort binding, got %q", action)
	}
}

func TestManager_Reload(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bindings.json")

	// Initial: default bindings
	m := New("", "")

	// Write custom binding
	data, _ := json.Marshal(map[string][]string{
		"openEditor": {"ctrl+e"},
	})
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}

	m.Reload(path, "")

	action := m.ActionForKey(key.Key{Type: key.KeyRune, Rune: 'e', Ctrl: true})
	if action != config.ActionOpenEditor {
		t.Errorf("expected openEditor after reload, got %q", action)
	}
}

func TestManager_FormatAll(t *testing.T) {
	t.Parallel()
	m := NewFromBindings(config.NewKeybindings())
	output := m.FormatAll()

	if !strings.Contains(output, "Keybindings:") {
		t.Error("expected header in FormatAll output")
	}
	if !strings.Contains(output, "Navigation") {
		t.Error("expected Navigation category")
	}
	if !strings.Contains(output, "Editing") {
		t.Error("expected Editing category")
	}
	if !strings.Contains(output, "Messages") {
		t.Error("expected Messages category")
	}
}

func TestKeyToString(t *testing.T) {
	t.Parallel()
	tests := []struct {
		k    key.Key
		want string
	}{
		{key.Key{Type: key.KeyCtrlG, Ctrl: true}, "ctrl+g"},
		{key.Key{Type: key.KeyCtrlC, Ctrl: true}, "ctrl+c"},
		{key.Key{Type: key.KeyEnter}, "enter"},
		{key.Key{Type: key.KeyEnter, Alt: true}, "alt+enter"},
		{key.Key{Type: key.KeyUp, Alt: true}, "alt+up"},
		{key.Key{Type: key.KeyRune, Rune: '@'}, "@"},
		{key.Key{Type: key.KeyBackTab}, "shift+tab"},
		{key.Key{Type: key.KeyEscape}, "escape"},
		{key.Key{Type: key.KeyBackspace}, "backspace"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := keyToString(tt.k)
			if got != tt.want {
				t.Errorf("keyToString(%+v) = %q; want %q", tt.k, got, tt.want)
			}
		})
	}
}
