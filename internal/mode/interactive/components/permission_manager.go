// ABOUTME: Permission manager component for managing permission rules
// ABOUTME: Allows users to view, add, and delete permission rules
// ABOUTME: Updated with Claude-style minimal border styling

package components

import (
	"fmt"
	"strings"
	"sync"

	"github.com/mauromedda/pi-coding-agent-go/internal/permission"
	"github.com/mauromedda/pi-coding-agent-go/pkg/tui"
	"github.com/mauromedda/pi-coding-agent-go/pkg/tui/key"
	"github.com/mauromedda/pi-coding-agent-go/pkg/tui/theme"
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

// Render writes the rule list into the buffer with Claude-style minimal borders.
func (pm *PermissionManager) Render(out *tui.RenderBuffer, w int) {
	pm.mu.Lock()
	rules := pm.rules
	selected := pm.selected
	scrollOff := pm.scrollOff
	maxHeight := pm.maxHeight
	pm.mu.Unlock()

	if len(rules) == 0 {
		out.WriteLine("")
		out.WriteLine("┌ No permission rules configured ┐")
		out.WriteLine("└────────────────────────────────┘")
		return
	}

	// Header
	out.WriteLine("")
	out.WriteLine("┌ Permission Rules ┐")
	out.WriteLine("├──────────────────┤")

	end := min(scrollOff+maxHeight, len(rules))

	for i := scrollOff; i < end; i++ {
		rule := rules[i]
		line := pm.formatRule(rule, w, i == selected)
		out.WriteLine(line)
	}

	// Footer
	out.WriteLine("└──────────────────┘")
}

func (pm *PermissionManager) formatRule(rule *RuleWrapper, w int, selected bool) string {
	p := theme.Current().Palette

	mode := p.Error.Apply("deny")
	if rule.Rule.Allow {
		mode = p.Success.Apply("allow")
	}

	line := fmt.Sprintf("  %s  %s", mode, rule.Tool)

	// Truncate to width
	line = strings.TrimSpace(line)
	if len(line) > w {
		line = line[:w-1] + "…"
	}

	if selected {
		line = p.Bold.Code() + p.Selection.Code() + line + "\x1b[0m"
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
