// ABOUTME: ModelSelectorModel is a Bubble Tea overlay for selecting an AI model
// ABOUTME: Returns ModelSelectedMsg on enter, ModelSelectorDismissMsg on esc

package btea

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// ModelEntry represents an AI model available for selection.
type ModelEntry struct {
	ID   string
	Name string
}

// ModelSelectedMsg is returned when the user selects a model.
type ModelSelectedMsg struct{ Model ModelEntry }

// ModelSelectorDismissMsg is returned when the user dismisses the selector.
type ModelSelectorDismissMsg struct{}

// ModelSelectorModel displays a list of models for selection.
// Implements tea.Model with value semantics.
type ModelSelectorModel struct {
	models   []ModelEntry
	selected int
	width    int
}

// NewModelSelectorModel creates a ModelSelectorModel with the given models.
func NewModelSelectorModel(models []ModelEntry) ModelSelectorModel {
	return ModelSelectorModel{
		models: models,
	}
}

// Init returns nil; no commands needed at startup.
func (m ModelSelectorModel) Init() tea.Cmd {
	return nil
}

// Update handles key messages for navigation, selection, and dismiss.
func (m ModelSelectorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyUp:
			m.moveUp()
		case tea.KeyDown:
			m.moveDown()
		case tea.KeyEnter:
			if len(m.models) == 0 {
				return m, nil
			}
			selected := m.models[m.selected]
			return m, func() tea.Msg { return ModelSelectedMsg{Model: selected} }
		case tea.KeyEsc:
			return m, func() tea.Msg { return ModelSelectorDismissMsg{} }
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
	}
	return m, nil
}

// View renders the model selector with header and selection indicator.
func (m ModelSelectorModel) View() string {
	s := Styles()
	var b strings.Builder

	// Header
	b.WriteString(s.Bold.Render("Select Model"))
	b.WriteByte('\n')

	if len(m.models) == 0 {
		b.WriteString(s.Muted.Render("  No models available"))
		return b.String()
	}

	for i, model := range m.models {
		b.WriteByte('\n')

		var prefix string
		if i == m.selected {
			prefix = "> "
		} else {
			prefix = "  "
		}

		line := fmt.Sprintf("%s%s (%s)", prefix, model.Name, model.ID)
		if i == m.selected {
			line = s.Bold.Render(s.Selection.Render(line))
		}
		b.WriteString(line)
	}

	return b.String()
}

// --- Internal helpers ---

func (m *ModelSelectorModel) moveUp() {
	if m.selected > 0 {
		m.selected--
	}
}

func (m *ModelSelectorModel) moveDown() {
	if m.selected < len(m.models)-1 {
		m.selected++
	}
}
