// ABOUTME: Tests for ModelSelectorModel overlay: model selection list
// ABOUTME: Verifies navigation, enter selects, esc dismisses, View output

package btea

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// Compile-time checks.
var (
	_ tea.Model = ModelSelectorModel{}
	_ tea.Msg   = ModelSelectedMsg{}
	_ tea.Msg   = ModelSelectorDismissMsg{}
)

func testModels() []ModelEntry {
	return []ModelEntry{
		{ID: "claude-opus-4", Name: "Opus"},
		{ID: "claude-sonnet-4", Name: "Sonnet"},
		{ID: "claude-haiku-3", Name: "Haiku"},
	}
}

func TestModelSelectorModel_Init(t *testing.T) {
	m := NewModelSelectorModel(testModels())
	if cmd := m.Init(); cmd != nil {
		t.Errorf("Init() returned non-nil cmd")
	}
}

func TestModelSelectorModel_Navigation(t *testing.T) {
	m := NewModelSelectorModel(testModels())

	// Initially selected = 0
	tests := []struct {
		name    string
		key     tea.KeyType
		wantID  string
	}{
		{"down to Sonnet", tea.KeyDown, "claude-sonnet-4"},
		{"down to Haiku", tea.KeyDown, "claude-haiku-3"},
		{"down at bottom stays", tea.KeyDown, "claude-haiku-3"},
		{"up to Sonnet", tea.KeyUp, "claude-sonnet-4"},
		{"up to Opus", tea.KeyUp, "claude-opus-4"},
		{"up at top stays", tea.KeyUp, "claude-opus-4"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updated, _ := m.Update(tea.KeyMsg{Type: tt.key})
			m = updated.(ModelSelectorModel)
			if m.models[m.selected].ID != tt.wantID {
				t.Errorf("selected model ID = %q; want %q", m.models[m.selected].ID, tt.wantID)
			}
		})
	}
}

func TestModelSelectorModel_EnterSelectsModel(t *testing.T) {
	m := NewModelSelectorModel(testModels())

	// Move down to Sonnet
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(ModelSelectorModel)

	// Press enter
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("Update(enter) returned nil cmd; want ModelSelectedMsg")
	}
	msg := cmd()
	sel, ok := msg.(ModelSelectedMsg)
	if !ok {
		t.Fatalf("cmd() returned %T; want ModelSelectedMsg", msg)
	}
	if sel.Model.ID != "claude-sonnet-4" {
		t.Errorf("ModelSelectedMsg.Model.ID = %q; want %q", sel.Model.ID, "claude-sonnet-4")
	}
	if sel.Model.Name != "Sonnet" {
		t.Errorf("ModelSelectedMsg.Model.Name = %q; want %q", sel.Model.Name, "Sonnet")
	}
}

func TestModelSelectorModel_EscDismisses(t *testing.T) {
	m := NewModelSelectorModel(testModels())

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("Update(esc) returned nil cmd; want ModelSelectorDismissMsg")
	}
	msg := cmd()
	if _, ok := msg.(ModelSelectorDismissMsg); !ok {
		t.Errorf("cmd() returned %T; want ModelSelectorDismissMsg", msg)
	}
}

func TestModelSelectorModel_ViewContainsHeader(t *testing.T) {
	m := NewModelSelectorModel(testModels())
	m.width = 80
	view := m.View()

	if !strings.Contains(view, "Select Model") {
		t.Error("View() missing 'Select Model' header")
	}
}

func TestModelSelectorModel_ViewShowsModels(t *testing.T) {
	m := NewModelSelectorModel(testModels())
	m.width = 80
	view := m.View()

	for _, model := range testModels() {
		if !strings.Contains(view, model.Name) {
			t.Errorf("View() missing model name %q", model.Name)
		}
	}
}

func TestModelSelectorModel_ViewSelectionIndicator(t *testing.T) {
	m := NewModelSelectorModel(testModels())
	m.width = 80
	view := m.View()

	if !strings.Contains(view, "> ") {
		t.Error("View() missing selection indicator '> '")
	}
}

func TestModelSelectorModel_ViewEmpty(t *testing.T) {
	m := NewModelSelectorModel(nil)
	m.width = 80
	view := m.View()

	// Should still render header at minimum
	if !strings.Contains(view, "Select Model") {
		t.Error("View() on empty list missing 'Select Model' header")
	}
}

func TestModelSelectorModel_EnterEmptyList(t *testing.T) {
	m := NewModelSelectorModel(nil)

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Errorf("Update(enter) on empty list returned non-nil cmd; want nil")
	}
}
