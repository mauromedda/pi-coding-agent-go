// ABOUTME: Tests for HookManagerModel overlay: hook list, toggle, remove, navigation
// ABOUTME: Verifies SetHooks, ToggleHook, RemoveHook, View output, and key handling

package btea

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// Compile-time check: HookManagerModel must satisfy tea.Model.
var _ tea.Model = HookManagerModel{}

func testHooks() []Hook {
	return []Hook{
		{Pattern: "*.go", Enabled: true, Tools: []string{"Read", "Write"}, Event: "pre-tool"},
		{Pattern: "*.py", Enabled: false, Tools: []string{"Bash"}, Event: "post-tool"},
		{Pattern: "*.md", Enabled: true, Tools: nil, Event: "pre-tool"},
	}
}

func TestHookManagerModel_Init(t *testing.T) {
	m := NewHookManagerModel()
	if cmd := m.Init(); cmd != nil {
		t.Errorf("Init() returned non-nil cmd")
	}
}

func TestHookManagerModel_SetHooks(t *testing.T) {
	m := NewHookManagerModel()
	hooks := testHooks()
	m = m.SetHooks(hooks)

	if m.Count() != 3 {
		t.Errorf("Count() = %d; want 3", m.Count())
	}
}

func TestHookManagerModel_CountEmpty(t *testing.T) {
	m := NewHookManagerModel()
	if m.Count() != 0 {
		t.Errorf("Count() on empty = %d; want 0", m.Count())
	}
}

func TestHookManagerModel_SelectedHook(t *testing.T) {
	m := NewHookManagerModel()
	m = m.SetHooks(testHooks())

	sel := m.SelectedHook()
	if sel.Pattern != "*.go" {
		t.Errorf("SelectedHook().Pattern = %q; want %q", sel.Pattern, "*.go")
	}
}

func TestHookManagerModel_SelectedHookEmpty(t *testing.T) {
	m := NewHookManagerModel()
	sel := m.SelectedHook()
	if sel.Pattern != "" {
		t.Errorf("SelectedHook() on empty = %+v; want zero value", sel)
	}
}

func TestHookManagerModel_Navigation(t *testing.T) {
	m := NewHookManagerModel()
	m = m.SetHooks(testHooks())

	// Move down
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(HookManagerModel)
	sel := m.SelectedHook()
	if sel.Pattern != "*.py" {
		t.Errorf("after down: Pattern = %q; want %q", sel.Pattern, "*.py")
	}

	// Move down
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(HookManagerModel)
	sel = m.SelectedHook()
	if sel.Pattern != "*.md" {
		t.Errorf("after 2x down: Pattern = %q; want %q", sel.Pattern, "*.md")
	}

	// Move down at bottom: stays
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(HookManagerModel)
	sel = m.SelectedHook()
	if sel.Pattern != "*.md" {
		t.Errorf("after down at bottom: Pattern = %q; want %q", sel.Pattern, "*.md")
	}

	// Move up
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = updated.(HookManagerModel)
	sel = m.SelectedHook()
	if sel.Pattern != "*.py" {
		t.Errorf("after up: Pattern = %q; want %q", sel.Pattern, "*.py")
	}
}

func TestHookManagerModel_ToggleHook(t *testing.T) {
	m := NewHookManagerModel()
	m = m.SetHooks(testHooks())

	// First hook is enabled; toggle should disable it
	m = m.ToggleHook()
	sel := m.SelectedHook()
	if sel.Enabled {
		t.Error("after ToggleHook: Enabled = true; want false")
	}

	// Toggle again: should re-enable
	m = m.ToggleHook()
	sel = m.SelectedHook()
	if !sel.Enabled {
		t.Error("after 2x ToggleHook: Enabled = false; want true")
	}
}

func TestHookManagerModel_ToggleHookViaEnter(t *testing.T) {
	m := NewHookManagerModel()
	m = m.SetHooks(testHooks())

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(HookManagerModel)
	sel := m.SelectedHook()
	if sel.Enabled {
		t.Error("after enter: Enabled = true; want false (toggled)")
	}
}

func TestHookManagerModel_RemoveHook(t *testing.T) {
	m := NewHookManagerModel()
	m = m.SetHooks(testHooks())

	// Remove first hook (*.go)
	m = m.RemoveHook()
	if m.Count() != 2 {
		t.Fatalf("after RemoveHook: Count() = %d; want 2", m.Count())
	}
	sel := m.SelectedHook()
	if sel.Pattern != "*.py" {
		t.Errorf("after RemoveHook: Pattern = %q; want %q", sel.Pattern, "*.py")
	}
}

func TestHookManagerModel_RemoveHookViaKeyD(t *testing.T) {
	m := NewHookManagerModel()
	m = m.SetHooks(testHooks())

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	m = updated.(HookManagerModel)
	if m.Count() != 2 {
		t.Errorf("after 'd' key: Count() = %d; want 2", m.Count())
	}
}

func TestHookManagerModel_RemoveHookViaDelete(t *testing.T) {
	m := NewHookManagerModel()
	m = m.SetHooks(testHooks())

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDelete})
	m = updated.(HookManagerModel)
	if m.Count() != 2 {
		t.Errorf("after delete key: Count() = %d; want 2", m.Count())
	}
}

func TestHookManagerModel_RemoveHookEmpty(t *testing.T) {
	m := NewHookManagerModel()
	m = m.RemoveHook() // should not panic
	if m.Count() != 0 {
		t.Errorf("RemoveHook on empty: Count() = %d; want 0", m.Count())
	}
}

func TestHookManagerModel_Reset(t *testing.T) {
	m := NewHookManagerModel()
	m = m.SetHooks(testHooks())

	// Navigate down
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(HookManagerModel)

	m = m.Reset()
	if m.Count() != 0 {
		t.Errorf("after Reset: Count() = %d; want 0", m.Count())
	}
}

func TestHookManagerModel_ViewContainsPattern(t *testing.T) {
	m := NewHookManagerModel()
	m = m.SetHooks(testHooks())
	m.width = 80

	view := m.View()
	if !strings.Contains(view, "*.go") {
		t.Error("View() missing pattern '*.go'")
	}
	if !strings.Contains(view, "*.py") {
		t.Error("View() missing pattern '*.py'")
	}
}

func TestHookManagerModel_ViewShowsEnabledDisabled(t *testing.T) {
	m := NewHookManagerModel()
	m = m.SetHooks(testHooks())
	m.width = 80

	view := m.View()
	// Should show enabled/disabled status indicators
	if !strings.Contains(view, "ON") && !strings.Contains(view, "OFF") {
		t.Errorf("View() missing enabled/disabled indicators; got:\n%s", view)
	}
}

func TestHookManagerModel_ViewShowsTools(t *testing.T) {
	m := NewHookManagerModel()
	m = m.SetHooks(testHooks())
	m.width = 80

	view := m.View()
	if !strings.Contains(view, "Read") {
		t.Error("View() missing tool name 'Read'")
	}
}

func TestHookManagerModel_ViewEmpty(t *testing.T) {
	m := NewHookManagerModel()
	m.width = 80
	view := m.View()
	if view != "" {
		t.Errorf("View() on empty = %q; want empty", view)
	}
}

func TestHookManagerModel_RemoveLastAdjustsSelected(t *testing.T) {
	m := NewHookManagerModel()
	m = m.SetHooks([]Hook{
		{Pattern: "a", Enabled: true},
		{Pattern: "b", Enabled: true},
	})

	// Navigate to last
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(HookManagerModel)

	// Remove last
	m = m.RemoveHook()
	if m.Count() != 1 {
		t.Fatalf("Count() = %d; want 1", m.Count())
	}
	sel := m.SelectedHook()
	if sel.Pattern != "a" {
		t.Errorf("after removing last: Pattern = %q; want %q", sel.Pattern, "a")
	}
}
