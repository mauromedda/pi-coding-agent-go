// ABOUTME: Keybindings manager with O(1) key-to-action lookup
// ABOUTME: Merges global and local configs, detects conflicts, supports hot-reload

package keybindings

import (
	"fmt"
	"maps"
	"strings"

	"github.com/mauromedda/pi-coding-agent-go/internal/config"
	"github.com/mauromedda/pi-coding-agent-go/pkg/tui/key"
)

// ConflictInfo describes a binding conflict where multiple actions share a key.
type ConflictInfo struct {
	Key     string
	Actions []config.KeyAction
}

// Manager provides O(1) key-to-action lookup from merged keybindings.
type Manager struct {
	bindings *config.Keybindings
	lookup   map[string]config.KeyAction // "ctrl+g" → ActionOpenEditor
}

// New creates a Manager from global and local keybinding files.
// Local bindings override global ones. Missing files are ignored.
func New(globalPath, localPath string) *Manager {
	kb := config.NewKeybindings()

	// Load global overrides
	if globalPath != "" {
		if g, err := config.LoadKeybindings(globalPath); err == nil {
			mergeBindings(kb, g)
		}
	}

	// Load local overrides (project takes precedence)
	if localPath != "" {
		if l, err := config.LoadKeybindings(localPath); err == nil {
			mergeBindings(kb, l)
		}
	}

	m := &Manager{bindings: kb}
	m.buildLookup()
	return m
}

// NewFromBindings creates a Manager from an existing Keybindings instance.
func NewFromBindings(kb *config.Keybindings) *Manager {
	m := &Manager{bindings: kb}
	m.buildLookup()
	return m
}

// ActionForKey returns the action bound to the given key, or "" if unbound.
func (m *Manager) ActionForKey(k key.Key) config.KeyAction {
	keyStr := keyToString(k)
	return m.lookup[keyStr]
}

// Conflicts detects keys bound to multiple actions.
func (m *Manager) Conflicts() []ConflictInfo {
	keyActions := make(map[string][]config.KeyAction)
	for action, keys := range m.bindings.Bindings {
		for _, k := range keys {
			keyActions[k] = append(keyActions[k], action)
		}
	}

	var conflicts []ConflictInfo
	for k, actions := range keyActions {
		if len(actions) > 1 {
			conflicts = append(conflicts, ConflictInfo{Key: k, Actions: actions})
		}
	}
	return conflicts
}

// Reload re-reads keybinding files and rebuilds the lookup table.
func (m *Manager) Reload(globalPath, localPath string) {
	kb := config.NewKeybindings()

	if globalPath != "" {
		if g, err := config.LoadKeybindings(globalPath); err == nil {
			mergeBindings(kb, g)
		}
	}
	if localPath != "" {
		if l, err := config.LoadKeybindings(localPath); err == nil {
			mergeBindings(kb, l)
		}
	}

	m.bindings = kb
	m.buildLookup()
}

// FormatAll returns a formatted table of all keybindings for /hotkeys display.
func (m *Manager) FormatAll() string {
	var b strings.Builder
	b.WriteString("Keybindings:\n\n")

	// Order by category for readability
	categories := []struct {
		name    string
		actions []config.KeyAction
	}{
		{"Navigation", []config.KeyAction{
			config.ActionCursorUp, config.ActionCursorDown,
			config.ActionCursorLeft, config.ActionCursorRight,
			config.ActionHome, config.ActionEnd,
		}},
		{"Editing", []config.KeyAction{
			config.ActionDeleteBack, config.ActionDeleteForward,
			config.ActionDeleteWordLeft, config.ActionDeleteLine,
			config.ActionPaste, config.ActionOpenEditor,
		}},
		{"Messages", []config.KeyAction{
			config.ActionSendMessage, config.ActionQueueFollowUp,
			config.ActionHistoryPrev, config.ActionHistoryNext,
			config.ActionFileMention,
		}},
		{"Scrolling", []config.KeyAction{
			config.ActionScrollUp, config.ActionScrollDown,
			config.ActionPageUp, config.ActionPageDown,
		}},
		{"Mode & Control", []config.KeyAction{
			config.ActionToggleMode, config.ActionAbort, config.ActionExit,
			config.ActionReload, config.ActionCycleModel,
			config.ActionToggleThinking, config.ActionToggleVim,
		}},
	}

	for _, cat := range categories {
		fmt.Fprintf(&b, "## %s\n", cat.name)
		for _, action := range cat.actions {
			keys := m.bindings.GetBindings(action)
			if len(keys) == 0 {
				continue
			}
			fmt.Fprintf(&b, "  %-20s %s\n", strings.Join(keys, ", "), action)
		}
		b.WriteString("\n")
	}

	return b.String()
}

func (m *Manager) buildLookup() {
	m.lookup = make(map[string]config.KeyAction, len(m.bindings.Bindings)*2)
	for action, keys := range m.bindings.Bindings {
		for _, k := range keys {
			m.lookup[k] = action
		}
	}
}

// mergeBindings overrides base bindings with overrides where present.
func mergeBindings(base, overrides *config.Keybindings) {
	maps.Copy(base.Bindings, overrides.Bindings)
}

// keyToString converts a key.Key to the string format used in keybinding configs.
func keyToString(k key.Key) string {
	var parts []string

	if k.Ctrl {
		parts = append(parts, "ctrl")
	}
	if k.Alt {
		parts = append(parts, "alt")
	}
	if k.Shift {
		parts = append(parts, "shift")
	}

	switch k.Type {
	case key.KeyRune:
		parts = append(parts, string(k.Rune))
	case key.KeyEnter:
		parts = append(parts, "enter")
	case key.KeyTab:
		parts = append(parts, "tab")
	case key.KeyBackTab:
		return "shift+tab" // special case: BackTab implies shift
	case key.KeyBackspace:
		parts = append(parts, "backspace")
	case key.KeyDelete:
		parts = append(parts, "delete")
	case key.KeyUp:
		parts = append(parts, "up")
	case key.KeyDown:
		parts = append(parts, "down")
	case key.KeyLeft:
		parts = append(parts, "left")
	case key.KeyRight:
		parts = append(parts, "right")
	case key.KeyHome:
		parts = append(parts, "home")
	case key.KeyEnd:
		parts = append(parts, "end")
	case key.KeyPageUp:
		parts = append(parts, "pgup")
	case key.KeyPageDown:
		parts = append(parts, "pgdown")
	case key.KeyEscape:
		parts = append(parts, "escape")
	case key.KeyCtrlC:
		return "ctrl+c"
	case key.KeyCtrlD:
		return "ctrl+d"
	case key.KeyCtrlG:
		return "ctrl+g"
	case key.KeyCtrlL:
		return "ctrl+l"
	case key.KeyCtrlO:
		return "ctrl+o"
	case key.KeyCtrlR:
		return "ctrl+r"
	default:
		return ""
	}

	return strings.Join(parts, "+")
}
