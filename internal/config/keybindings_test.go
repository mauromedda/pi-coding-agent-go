// ABOUTME: Tests for keybindings parser

package config

import (
	"encoding/json"
	"path/filepath"
	"testing"
)

func TestKeybindings_New(t *testing.T) {
	kb := NewKeybindings()
	if kb == nil {
		t.Fatal("NewKeybindings returned nil")
	}
	if len(kb.Bindings) == 0 {
		t.Error("Expected default bindings")
	}
}

func TestKeybindings_SetDefaultBindings(t *testing.T) {
	kb := NewKeybindings()

	// Check some default bindings
	if len(kb.Bindings[ActionCursorUp]) == 0 {
		t.Error("Expected cursorUp bindings")
	}
	if len(kb.Bindings[ActionPaste]) == 0 {
		t.Error("Expected paste bindings")
	}
	if len(kb.Bindings[ActionSendMessageAlt]) == 0 {
		t.Error("Expected sendMessageAlt bindings")
	}
}

func TestKeybindings_SaveLoad(t *testing.T) {
	kb := NewKeybindings()
	kb.Bindings[ActionCursorUp] = []string{"up", "ctrl+k"}
	kb.Bindings[ActionPaste] = []string{"ctrl+v"}

	// Create temp file
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "keybindings.json")

	// Save
	if err := kb.SaveKeybindings(path); err != nil {
		t.Fatalf("SaveKeybindings failed: %v", err)
	}

	// Load
	loaded, err := LoadKeybindings(path)
	if err != nil {
		t.Fatalf("LoadKeybindings failed: %v", err)
	}

	// Check
	if len(loaded.Bindings[ActionCursorUp]) != 2 {
		t.Errorf("Expected 2 cursorUp bindings, got %d", len(loaded.Bindings[ActionCursorUp]))
	}
}

func TestKeybindings_GetBindings(t *testing.T) {
	kb := NewKeybindings()

	bindings := kb.GetBindings(ActionCursorUp)
	if len(bindings) == 0 {
		t.Error("Expected bindings for cursorUp")
	}
}

func TestKeybindings_GlobalPath(t *testing.T) {
	path := GlobalKeybindingsFile()
	if path == "" {
		t.Error("Expected non-empty path")
	}
	if !filepath.IsAbs(path) {
		t.Errorf("Expected absolute path, got %s", path)
	}
}

func TestKeybindings_LocalPath(t *testing.T) {
	path := LocalKeybindingsFile("/test/project")
	expected := filepath.Join("/test/project", ".pi-go", "agent", "keybindings.json")
	if path != expected {
		t.Errorf("Expected %s, got %s", expected, path)
	}
}

func TestKeybindings_LoadNonExistent(t *testing.T) {
	_, err := LoadKeybindings("/nonexistent/path/keybindings.json")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestKeybindings_CustomBindings(t *testing.T) {
	kb := NewKeybindings()

	// Override some bindings
	kb.Bindings[ActionCursorUp] = []string{"ctrl+k"}
	kb.Bindings[ActionCursorDown] = []string{"ctrl+j"}

	if len(kb.Bindings[ActionCursorUp]) != 1 {
		t.Error("Expected custom cursorUp binding")
	}
}

func TestKeybindings_NoDuplicateDefaults(t *testing.T) {
	t.Parallel()

	kb := NewKeybindings()

	// Build reverse map: key combo → list of actions bound to it
	reverseMap := make(map[string][]KeyAction)
	for action, keys := range kb.Bindings {
		for _, k := range keys {
			reverseMap[k] = append(reverseMap[k], action)
		}
	}

	// "enter" is intentionally shared by accept and sendMessage
	// (same physical key, different semantic contexts).
	allowedOverlaps := map[string]bool{"enter": true}

	for combo, actions := range reverseMap {
		if len(actions) > 1 && !allowedOverlaps[combo] {
			t.Errorf("key %q is bound to multiple actions: %v", combo, actions)
		}
	}
}

func TestKeybindings_RawJSON(t *testing.T) {
	raw := RawKeybindings{
		"cursorUp":   []string{"up", "ctrl+p"},
		"cursorDown": []string{"down", "ctrl+n"},
	}

	data, err := json.Marshal(raw)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var loaded RawKeybindings
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if len(loaded["cursorUp"]) != 2 {
		t.Errorf("Expected 2 bindings, got %d", len(loaded["cursorUp"]))
	}
}
