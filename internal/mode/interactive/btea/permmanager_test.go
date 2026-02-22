// ABOUTME: Tests for PermManagerModel overlay: permission rules list, navigation, remove
// ABOUTME: Verifies SetRules, RemoveRule, View output with allow/deny color coding

package btea

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// Compile-time check: PermManagerModel must satisfy tea.Model.
var _ tea.Model = PermManagerModel{}

func testRules() []RuleEntry {
	return []RuleEntry{
		{Tool: "Bash", Allow: true},
		{Tool: "Write", Allow: false},
		{Tool: "Read", Allow: true},
	}
}

func TestPermManagerModel_Init(t *testing.T) {
	m := NewPermManagerModel()
	if cmd := m.Init(); cmd != nil {
		t.Errorf("Init() returned non-nil cmd")
	}
}

func TestPermManagerModel_SetRules(t *testing.T) {
	m := NewPermManagerModel()
	m = m.SetRules(testRules())
	if m.Count() != 3 {
		t.Errorf("Count() = %d; want 3", m.Count())
	}
}

func TestPermManagerModel_CountEmpty(t *testing.T) {
	m := NewPermManagerModel()
	if m.Count() != 0 {
		t.Errorf("Count() on empty = %d; want 0", m.Count())
	}
}

func TestPermManagerModel_SelectedRule(t *testing.T) {
	m := NewPermManagerModel()
	m = m.SetRules(testRules())

	sel := m.SelectedRule()
	if sel.Tool != "Bash" {
		t.Errorf("SelectedRule().Tool = %q; want %q", sel.Tool, "Bash")
	}
}

func TestPermManagerModel_SelectedRuleEmpty(t *testing.T) {
	m := NewPermManagerModel()
	sel := m.SelectedRule()
	if sel.Tool != "" {
		t.Errorf("SelectedRule() on empty = %+v; want zero value", sel)
	}
}

func TestPermManagerModel_Navigation(t *testing.T) {
	m := NewPermManagerModel()
	m = m.SetRules(testRules())

	// Move down
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(PermManagerModel)
	sel := m.SelectedRule()
	if sel.Tool != "Write" {
		t.Errorf("after down: Tool = %q; want %q", sel.Tool, "Write")
	}

	// Move down
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(PermManagerModel)
	sel = m.SelectedRule()
	if sel.Tool != "Read" {
		t.Errorf("after 2x down: Tool = %q; want %q", sel.Tool, "Read")
	}

	// Move down at bottom: stays
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(PermManagerModel)
	sel = m.SelectedRule()
	if sel.Tool != "Read" {
		t.Errorf("after down at bottom: Tool = %q; want %q", sel.Tool, "Read")
	}

	// Move up
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = updated.(PermManagerModel)
	sel = m.SelectedRule()
	if sel.Tool != "Write" {
		t.Errorf("after up: Tool = %q; want %q", sel.Tool, "Write")
	}
}

func TestPermManagerModel_RemoveRule(t *testing.T) {
	m := NewPermManagerModel()
	m = m.SetRules(testRules())

	// Remove first rule (Bash)
	m = m.RemoveRule()
	if m.Count() != 2 {
		t.Fatalf("after RemoveRule: Count() = %d; want 2", m.Count())
	}
	sel := m.SelectedRule()
	if sel.Tool != "Write" {
		t.Errorf("after RemoveRule: Tool = %q; want %q", sel.Tool, "Write")
	}
}

func TestPermManagerModel_RemoveRuleViaKeyD(t *testing.T) {
	m := NewPermManagerModel()
	m = m.SetRules(testRules())

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	m = updated.(PermManagerModel)
	if m.Count() != 2 {
		t.Errorf("after 'd' key: Count() = %d; want 2", m.Count())
	}
}

func TestPermManagerModel_RemoveRuleViaDelete(t *testing.T) {
	m := NewPermManagerModel()
	m = m.SetRules(testRules())

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDelete})
	m = updated.(PermManagerModel)
	if m.Count() != 2 {
		t.Errorf("after delete key: Count() = %d; want 2", m.Count())
	}
}

func TestPermManagerModel_RemoveRuleEmpty(t *testing.T) {
	m := NewPermManagerModel()
	m = m.RemoveRule() // should not panic
	if m.Count() != 0 {
		t.Errorf("RemoveRule on empty: Count() = %d; want 0", m.Count())
	}
}

func TestPermManagerModel_Reset(t *testing.T) {
	m := NewPermManagerModel()
	m = m.SetRules(testRules())

	m = m.Reset()
	if m.Count() != 0 {
		t.Errorf("after Reset: Count() = %d; want 0", m.Count())
	}
}

func TestPermManagerModel_ViewContainsToolName(t *testing.T) {
	m := NewPermManagerModel()
	m = m.SetRules(testRules())
	m.width = 80

	view := m.View()
	if !strings.Contains(view, "Bash") {
		t.Error("View() missing tool name 'Bash'")
	}
	if !strings.Contains(view, "Write") {
		t.Error("View() missing tool name 'Write'")
	}
}

func TestPermManagerModel_ViewShowsAllowDeny(t *testing.T) {
	m := NewPermManagerModel()
	m = m.SetRules(testRules())
	m.width = 80

	view := m.View()
	if !strings.Contains(view, "ALLOW") && !strings.Contains(view, "DENY") {
		t.Errorf("View() missing ALLOW/DENY indicators; got:\n%s", view)
	}
}

func TestPermManagerModel_ViewEmpty(t *testing.T) {
	m := NewPermManagerModel()
	m.width = 80
	view := m.View()
	if view != "" {
		t.Errorf("View() on empty = %q; want empty", view)
	}
}

func TestPermManagerModel_RemoveLastAdjustsSelected(t *testing.T) {
	m := NewPermManagerModel()
	m = m.SetRules([]RuleEntry{
		{Tool: "A", Allow: true},
		{Tool: "B", Allow: true},
	})

	// Navigate to last
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(PermManagerModel)

	// Remove last
	m = m.RemoveRule()
	if m.Count() != 1 {
		t.Fatalf("Count() = %d; want 1", m.Count())
	}
	sel := m.SelectedRule()
	if sel.Tool != "A" {
		t.Errorf("after removing last: Tool = %q; want %q", sel.Tool, "A")
	}
}
