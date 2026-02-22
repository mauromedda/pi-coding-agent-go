// ABOUTME: Permission manager component for managing permission rules
// ABOUTME: Allows users to view, add, and delete permission rules

package components

import (
	"fmt"
	"strings"
	"sync"

	"github.com/mauromedda/pi-coding-agent-go/internal/permission"
	"github.com/mauromedda/pi-coding-agent-go/pkg/tui"
	"github.com/mauromedda/pi-coding-agent-go/pkg/tui/key"
)

// RuleWrapper wraps a permission.Rule for display
type RuleWrapper struct {
	Rule *permission.Rule
	Tool string // Extracted tool name
	Mode string // allow/deny mode
}

// PermissionManager manages permission rules
type PermissionManager struct {
	rules     []*RuleWrapper
	selected  int
	scrollOff int
	maxHeight int
	dirty     bool
	mu        sync.Mutex
}

// NewPermissionManager creates a new permission manager
func NewPermissionManager() *PermissionManager {
	return &PermissionManager{
		maxHeight: 20,
		dirty:     true,
	}
}

// SetRules sets the rules to display
func (pm *PermissionManager) SetRules(rules []*permission.Rule) {
	pm.mu.Lock()
	pm.rules = make([]*RuleWrapper, len(rules))
	for i, r := range rules {
		pm.rules[i] = &RuleWrapper{
			Rule: r,
			Tool: r.Tool,
			Mode: "allow",
		}
		if !r.Allow {
			pm.rules[i].Mode = "deny"
		}
	}
	pm.selected = 0
	pm.scrollOff = 0
	pm.dirty = true
	pm.mu.Unlock()
}

// AddRule adds a new rule
func (pm *PermissionManager) AddRule(rule *permission.Rule) {
	pm.mu.Lock()
	pm.rules = append(pm.rules, &RuleWrapper{
		Rule: rule,
		Tool: rule.Tool,
		Mode: "allow",
	})
	if !rule.Allow {
		pm.rules[len(pm.rules)-1].Mode = "deny"
	}
	pm.selected = len(pm.rules) - 1
	pm.dirty = true
	pm.mu.Unlock()
}

// RemoveRule removes a rule
func (pm *PermissionManager) RemoveRule() {
	pm.mu.Lock()
	if pm.selected < 0 || pm.selected >= len(pm.rules) {
		pm.mu.Unlock()
		return
	}
	pm.rules = append(pm.rules[:pm.selected], pm.rules[pm.selected+1:]...)
	if pm.selected >= len(pm.rules) {
		pm.selected = len(pm.rules) - 1
	}
	pm.dirty = true
	pm.mu.Unlock()
}

// SelectedRule returns the currently selected rule
func (pm *PermissionManager) SelectedRule() *RuleWrapper {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	if pm.selected < 0 || pm.selected >= len(pm.rules) {
		return nil
	}
	return pm.rules[pm.selected]
}

// Invalidate marks the component for re-render
func (pm *PermissionManager) Invalidate() {
	pm.mu.Lock()
	pm.dirty = true
	pm.mu.Unlock()
}

// HandleKey processes a parsed key event for navigation
func (pm *PermissionManager) HandleKey(k key.Key) {
	pm.mu.Lock()
	switch k.Type {
	case key.KeyUp:
		pm.moveUpLocked()
	case key.KeyDown:
		pm.moveDownLocked()
	case key.KeyEnter:
		pm.dirty = true
	}
	pm.mu.Unlock()
}

func (pm *PermissionManager) moveUp() {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.moveUpLocked()
}

func (pm *PermissionManager) moveUpLocked() {
	if pm.selected > 0 {
		pm.selected--
		pm.adjustScrollLocked()
		pm.dirty = true
	}
}

func (pm *PermissionManager) moveDown() {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.moveDownLocked()
}

func (pm *PermissionManager) moveDownLocked() {
	if pm.selected < len(pm.rules)-1 {
		pm.selected++
		pm.adjustScrollLocked()
		pm.dirty = true
	}
}

func (pm *PermissionManager) adjustScroll() {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.adjustScrollLocked()
}

func (pm *PermissionManager) adjustScrollLocked() {
	if pm.selected < pm.scrollOff {
		pm.scrollOff = pm.selected
	}
	if pm.selected >= pm.scrollOff+pm.maxHeight {
		pm.scrollOff = pm.selected - pm.maxHeight + 1
	}
}

// Render writes the rule list into the buffer
func (pm *PermissionManager) Render(out *tui.RenderBuffer, w int) {
	if len(pm.rules) == 0 {
		out.WriteLine("\x1b[2mNo permission rules configured\x1b[0m")
		return
	}

	end := min(pm.scrollOff+pm.maxHeight, len(pm.rules))

	for i := pm.scrollOff; i < end; i++ {
		rule := pm.rules[i]
		line := pm.formatRule(rule, w, i == pm.selected)
		out.WriteLine(line)
	}
}

func (pm *PermissionManager) formatRule(rule *RuleWrapper, w int, selected bool) string {
	mode := "\x1b[31mdeny\x1b[0m"
	if rule.Rule.Allow {
		mode = "\x1b[32mallow\x1b[0m"
	}

	line := fmt.Sprintf("  %s  %s", mode, rule.Tool)

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

// Count returns the number of rules
func (pm *PermissionManager) Count() int {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	return len(pm.rules)
}

// Reset clears the selection
func (pm *PermissionManager) Reset() {
	pm.mu.Lock()
	pm.selected = 0
	pm.scrollOff = 0
	pm.dirty = true
	pm.mu.Unlock()
}
