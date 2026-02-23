// ABOUTME: Tests for model selector overlay wiring
// ABOUTME: Verifies Ctrl+M opens selector, model switch updates deps and footer

package btea

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mauromedda/pi-coding-agent-go/pkg/ai"
)

func TestAppModel_CtrlMOpensModelSelector(t *testing.T) {
	deps := testDeps()
	deps.AvailableModels = []ModelEntry{
		{ID: "claude-3", Name: "Claude 3"},
		{ID: "gpt-4", Name: "GPT-4"},
	}
	m := NewAppModel(deps)

	key := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}, Alt: true}
	result, _ := m.Update(key)
	model := result.(AppModel)

	if model.overlay == nil {
		t.Fatal("overlay = nil; want ModelSelectorModel")
	}
	if _, ok := model.overlay.(ModelSelectorModel); !ok {
		t.Errorf("overlay = %T; want ModelSelectorModel", model.overlay)
	}
}

func TestAppModel_CtrlMNoModels(t *testing.T) {
	m := NewAppModel(testDeps()) // no AvailableModels

	key := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}, Alt: true}
	result, _ := m.Update(key)
	model := result.(AppModel)

	// Should still open overlay (shows "no models available")
	if model.overlay == nil {
		t.Fatal("overlay = nil; want ModelSelectorModel even with no models")
	}
}

func TestAppModel_ModelSelectedMsg_SwitchesModel(t *testing.T) {
	deps := testDeps()
	deps.AvailableModels = []ModelEntry{
		{ID: "claude-3", Name: "Claude 3"},
		{ID: "gpt-4", Name: "GPT-4"},
	}
	m := NewAppModel(deps)
	m.overlay = NewModelSelectorModel(deps.AvailableModels)

	// Select GPT-4
	result, _ := m.Update(ModelSelectedMsg{Model: ModelEntry{ID: "gpt-4", Name: "GPT-4"}})
	model := result.(AppModel)

	if model.overlay != nil {
		t.Error("overlay should be nil after ModelSelectedMsg")
	}

	// Model name should be updated in deps
	if model.deps.Model.Name != "GPT-4" {
		t.Errorf("model name = %q; want %q", model.deps.Model.Name, "GPT-4")
	}
	if model.deps.Model.ID != "gpt-4" {
		t.Errorf("model ID = %q; want %q", model.deps.Model.ID, "gpt-4")
	}
}

func TestAppModel_ModelSelectedMsg_UpdatesFooter(t *testing.T) {
	deps := testDeps()
	deps.AvailableModels = []ModelEntry{
		{ID: "gpt-4", Name: "GPT-4"},
	}
	m := NewAppModel(deps)

	result, _ := m.Update(ModelSelectedMsg{Model: ModelEntry{ID: "gpt-4", Name: "GPT-4"}})
	model := result.(AppModel)

	if model.footer.model != "GPT-4" {
		t.Errorf("footer model = %q; want %q", model.footer.model, "GPT-4")
	}
}

func TestAppModel_ModelSelectorDismissMsg(t *testing.T) {
	deps := testDeps()
	deps.AvailableModels = []ModelEntry{{ID: "m1", Name: "M1"}}
	m := NewAppModel(deps)
	m.overlay = NewModelSelectorModel(deps.AvailableModels)
	originalModel := m.deps.Model.Name

	result, _ := m.Update(ModelSelectorDismissMsg{})
	model := result.(AppModel)

	if model.overlay != nil {
		t.Error("overlay should be nil after dismiss")
	}
	// Model should NOT have changed
	if model.deps.Model.Name != originalModel {
		t.Errorf("model changed after dismiss: got %q; want %q", model.deps.Model.Name, originalModel)
	}
}

func TestAppDeps_AvailableModelsField(t *testing.T) {
	deps := AppDeps{
		Model: &ai.Model{Name: "test"},
		AvailableModels: []ModelEntry{
			{ID: "a", Name: "A"},
			{ID: "b", Name: "B"},
		},
	}
	if len(deps.AvailableModels) != 2 {
		t.Errorf("AvailableModels = %d; want 2", len(deps.AvailableModels))
	}
}
