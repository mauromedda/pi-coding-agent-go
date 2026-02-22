// ABOUTME: Hook manager component for viewing and managing hooks
// ABOUTME: Allows users to add, remove, enable, and disable hooks

package components

import (
	"fmt"
	"strings"
	"sync"

	"github.com/mauromedda/pi-coding-agent-go/internal/config"
	"github.com/mauromedda/pi-coding-agent-go/pkg/tui"
	"github.com/mauromedda/pi-coding-agent-go/pkg/tui/key"
)

// ConvertFromConfig maps config.HookDef entries to Hook structs.
func ConvertFromConfig(hooksByEvent map[string][]config.HookDef) []*Hook {
	var hooks []*Hook
	for event, defs := range hooksByEvent {
		for _, def := range defs {
			hooks = append(hooks, &Hook{
				Pattern: def.Matcher,
				Enabled: true,
				Event:   event,
			})
		}
	}
	return hooks
}

// Hook represents a hook configuration
type Hook struct {
	Pattern string   // Pattern to match (regex)
	Enabled bool     // Is hook enabled
	Tools   []string // Tools to hook
	Event   string   // Hook event type
}

// HookManager manages hook configurations
type HookManager struct {
	hooks     []*Hook
	selected  int
	scrollOff int
	maxHeight int
	dirty     bool
	mu        sync.Mutex
}

// NewHookManager creates a new hook manager
func NewHookManager() *HookManager {
	return &HookManager{
		maxHeight: 20,
		dirty:     true,
	}
}

// SetHooks sets the hooks to display
func (hm *HookManager) SetHooks(hooks []*Hook) {
	hm.mu.Lock()
	hm.hooks = hooks
	hm.selected = 0
	hm.scrollOff = 0
	hm.dirty = true
	hm.mu.Unlock()
}

// AddHook adds a new hook
func (hm *HookManager) AddHook(hook *Hook) {
	hm.mu.Lock()
	hm.hooks = append(hm.hooks, hook)
	hm.selected = len(hm.hooks) - 1
	hm.dirty = true
	hm.mu.Unlock()
}

// RemoveHook removes a hook
func (hm *HookManager) RemoveHook() {
	hm.mu.Lock()
	if hm.selected < 0 || hm.selected >= len(hm.hooks) {
		hm.mu.Unlock()
		return
	}
	hm.hooks = append(hm.hooks[:hm.selected], hm.hooks[hm.selected+1:]...)
	if hm.selected >= len(hm.hooks) {
		hm.selected = len(hm.hooks) - 1
	}
	hm.dirty = true
	hm.mu.Unlock()
}

// ToggleHook enables or disables the selected hook
func (hm *HookManager) ToggleHook() {
	hm.mu.Lock()
	if hm.selected < 0 || hm.selected >= len(hm.hooks) {
		hm.mu.Unlock()
		return
	}
	hm.hooks[hm.selected].Enabled = !hm.hooks[hm.selected].Enabled
	hm.dirty = true
	hm.mu.Unlock()
}

// SelectedHook returns the currently selected hook
func (hm *HookManager) SelectedHook() *Hook {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	if hm.selected < 0 || hm.selected >= len(hm.hooks) {
		return nil
	}
	return hm.hooks[hm.selected]
}

// Invalidate marks the component for re-render
func (hm *HookManager) Invalidate() {
	hm.mu.Lock()
	hm.dirty = true
	hm.mu.Unlock()
}

// HandleKey processes a parsed key event for navigation
func (hm *HookManager) HandleKey(k key.Key) {
	hm.mu.Lock()
	switch k.Type {
	case key.KeyUp:
		hm.moveUpLocked()
	case key.KeyDown:
		hm.moveDownLocked()
	case key.KeyEnter:
		hm.ToggleHookLocked()
		hm.dirty = true
	}
	hm.mu.Unlock()
}

func (hm *HookManager) moveUp() {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	hm.moveUpLocked()
}

func (hm *HookManager) moveUpLocked() {
	if hm.selected > 0 {
		hm.selected--
		hm.adjustScrollLocked()
		hm.dirty = true
	}
}

func (hm *HookManager) moveDown() {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	hm.moveDownLocked()
}

func (hm *HookManager) moveDownLocked() {
	if hm.selected < len(hm.hooks)-1 {
		hm.selected++
		hm.adjustScrollLocked()
		hm.dirty = true
	}
}

func (hm *HookManager) adjustScroll() {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	hm.adjustScrollLocked()
}

func (hm *HookManager) adjustScrollLocked() {
	if hm.selected < hm.scrollOff {
		hm.scrollOff = hm.selected
	}
	if hm.selected >= hm.scrollOff+hm.maxHeight {
		hm.scrollOff = hm.selected - hm.maxHeight + 1
	}
}

func (hm *HookManager) ToggleHookLocked() {
	if hm.selected < 0 || hm.selected >= len(hm.hooks) {
		return
	}
	hm.hooks[hm.selected].Enabled = !hm.hooks[hm.selected].Enabled
	hm.dirty = true
}

// Render writes the hook list into the buffer
func (hm *HookManager) Render(out *tui.RenderBuffer, w int) {
	if len(hm.hooks) == 0 {
		out.WriteLine("\x1b[2mNo hooks configured\x1b[0m")
		return
	}

	end := min(hm.scrollOff+hm.maxHeight, len(hm.hooks))

	for i := hm.scrollOff; i < end; i++ {
		hook := hm.hooks[i]
		line := hm.formatHook(hook, w, i == hm.selected)
		out.WriteLine(line)
	}
}

func (hm *HookManager) formatHook(hook *Hook, w int, selected bool) string {
	status := "\x1b[2mdisabled\x1b[0m"
	if hook.Enabled {
		status = "\x1b[32menabled\x1b[0m"
	}

	line := fmt.Sprintf("  %s  %s", status, hook.Pattern)

	// Add tools if present
	if len(hook.Tools) > 0 {
		tools := strings.Join(hook.Tools, ", ")
		line += fmt.Sprintf(" \x1b[90m[%s]\x1b[0m", tools)
	}

	// Truncate to width
	line = strings.TrimSpace(line)
	if len(line) > w {
		line = line[:w-1] + "…"
	}

	if selected {
		line = "\x1b[1m\x1b[7m" + line + "\x1b[0m"
	}

	return line
}

// Count returns the number of hooks
func (hm *HookManager) Count() int {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	return len(hm.hooks)
}

// Reset clears the selection
func (hm *HookManager) Reset() {
	hm.mu.Lock()
	hm.selected = 0
	hm.scrollOff = 0
	hm.dirty = true
	hm.mu.Unlock()
}
