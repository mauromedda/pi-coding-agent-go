// ABOUTME: Keybindings parser and loader for pi keybinding format
// ABOUTME: Supports ~/.pi-go/agent/keybindings.json format

package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// KeyAction represents an action that can be bound to keys
type KeyAction string

const (
	ActionCursorUp       KeyAction = "cursorUp"
	ActionCursorDown     KeyAction = "cursorDown"
	ActionCursorLeft     KeyAction = "cursorLeft"
	ActionCursorRight    KeyAction = "cursorRight"
	ActionDeleteBack     KeyAction = "deleteBack"
	ActionDeleteForward  KeyAction = "deleteForward"
	ActionDeleteWordLeft KeyAction = "deleteWordLeft"
	ActionDeleteLine     KeyAction = "deleteLine"
	ActionPaste          KeyAction = "paste"
	ActionAccept         KeyAction = "accept"
	ActionCancel         KeyAction = "cancel"
	ActionHistoryUp      KeyAction = "historyUp"
	ActionHistoryDown    KeyAction = "historyDown"
	ActionToggleMode     KeyAction = "toggleMode"
	ActionSendMessage    KeyAction = "sendMessage"
	ActionSendMessageAlt KeyAction = "sendMessageAlt"
	ActionScrollUp       KeyAction = "scrollUp"
	ActionScrollDown     KeyAction = "scrollDown"
	ActionPageUp         KeyAction = "pageUp"
	ActionPageDown       KeyAction = "pageDown"
	ActionHome           KeyAction = "home"
	ActionEnd            KeyAction = "end"
	ActionSearchForward  KeyAction = "searchForward"
	ActionSearchBack     KeyAction = "searchBack"
	ActionToggleThinking KeyAction = "toggleThinking"
	ActionCycleModel     KeyAction = "cycleModel"
	ActionToggleVim      KeyAction = "toggleVim"
	ActionReload         KeyAction = "reload"
)

// Keybindings represents the keybindings configuration
type Keybindings struct {
	Bindings map[KeyAction][]string `json:"-"`
}

// RawKeybindings is for JSON marshaling
type RawKeybindings map[string][]string

// NewKeybindings creates a new Keybindings with default bindings
func NewKeybindings() *Keybindings {
	kb := &Keybindings{
		Bindings: make(map[KeyAction][]string),
	}
	kb.setDefaultBindings()
	return kb
}

// setDefaultBindings sets default keybindings matching pi format
func (kb *Keybindings) setDefaultBindings() {
	kb.Bindings[ActionCursorUp] = []string{"up", "ctrl+p"}
	kb.Bindings[ActionCursorDown] = []string{"down", "ctrl+n"}
	kb.Bindings[ActionCursorLeft] = []string{"left", "ctrl+b"}
	kb.Bindings[ActionCursorRight] = []string{"right", "ctrl+f"}
	kb.Bindings[ActionDeleteBack] = []string{"backspace"}
	kb.Bindings[ActionDeleteForward] = []string{"delete"}
	kb.Bindings[ActionDeleteWordLeft] = []string{"ctrl+w"}
	kb.Bindings[ActionDeleteLine] = []string{"ctrl+k"}
	kb.Bindings[ActionPaste] = []string{"ctrl+v"}
	kb.Bindings[ActionAccept] = []string{"enter"}
	kb.Bindings[ActionCancel] = []string{"escape"}
	kb.Bindings[ActionHistoryUp] = []string{"alt+p"}
	kb.Bindings[ActionHistoryDown] = []string{"alt+n"}
	kb.Bindings[ActionToggleMode] = []string{"shift+tab"}
	kb.Bindings[ActionSendMessage] = []string{"enter"}
	kb.Bindings[ActionSendMessageAlt] = []string{"alt+enter"}
	kb.Bindings[ActionScrollUp] = []string{"pgup"}
	kb.Bindings[ActionScrollDown] = []string{"pgdown"}
	kb.Bindings[ActionPageUp] = []string{"shift+pgup"}
	kb.Bindings[ActionPageDown] = []string{"shift+pgdown"}
	kb.Bindings[ActionHome] = []string{"home"}
	kb.Bindings[ActionEnd] = []string{"end"}
	kb.Bindings[ActionToggleThinking] = []string{"alt+t"}
	kb.Bindings[ActionCycleModel] = []string{"shift+ctrl+p"}
	kb.Bindings[ActionToggleVim] = []string{"ctrl+@"}
	kb.Bindings[ActionReload] = []string{"ctrl+r"}
}

// LoadKeybindings loads keybindings from a file
func LoadKeybindings(path string) (*Keybindings, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var raw RawKeybindings
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	kb := NewKeybindings()
	for actionName, keys := range raw {
		action := KeyAction(actionName)
		if _, ok := kb.Bindings[action]; ok {
			kb.Bindings[action] = keys
		}
	}

	return kb, nil
}

// SaveKeybindings saves keybindings to a file
func (kb *Keybindings) SaveKeybindings(path string) error {
	raw := make(RawKeybindings)
	for action, keys := range kb.Bindings {
		raw[string(action)] = keys
	}

	data, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o644)
}

// GetBindings returns the bindings for an action
func (kb *Keybindings) GetBindings(action KeyAction) []string {
	if kb == nil {
		return nil
	}
	return kb.Bindings[action]
}

// GlobalKeybindingsFile returns the path to the global keybindings file
func GlobalKeybindingsFile() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".pi-go", "agent", "keybindings.json")
}

// LocalKeybindingsFile returns the path to the local keybindings file
func LocalKeybindingsFile(projectRoot string) string {
	return filepath.Join(projectRoot, ".pi-go", "agent", "keybindings.json")
}

// ExportTemplate exports current keybindings as a JSON template
func (kb *Keybindings) ExportTemplate() (string, error) {
	raw := make(RawKeybindings)
	for action, keys := range kb.Bindings {
		raw[string(action)] = keys
	}

	data, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}
