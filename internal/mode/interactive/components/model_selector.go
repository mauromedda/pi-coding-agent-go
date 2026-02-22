// ABOUTME: Model picker overlay for switching between LLM models
// ABOUTME: Displays a filterable list of available models

package components

import (
	"github.com/mauromedda/pi-coding-agent-go/pkg/ai"
	"github.com/mauromedda/pi-coding-agent-go/pkg/tui"
	"github.com/mauromedda/pi-coding-agent-go/pkg/tui/theme"
)

// ModelSelector displays a list of models to choose from.
type ModelSelector struct {
	models   []ai.Model
	selected int
	onSelect func(ai.Model)
}

// NewModelSelector creates a model picker overlay.
func NewModelSelector(models []ai.Model, onSelect func(ai.Model)) *ModelSelector {
	return &ModelSelector{
		models:   models,
		onSelect: onSelect,
	}
}

// Render draws the model list.
func (s *ModelSelector) Render(out *tui.RenderBuffer, _ int) {
	p := theme.Current().Palette
	out.WriteLine(p.Bold.Apply("  Select Model  "))
	out.WriteLine("")

	for i, m := range s.models {
		prefix := "  "
		if i == s.selected {
			prefix = p.Selection.Code() + "> " // Inverted for selection
		}
		line := prefix + m.Name + " (" + m.ID + ")"
		if i == s.selected {
			line += "\x1b[0m"
		}
		out.WriteLine(line)
	}
}

// Invalidate is a no-op.
func (s *ModelSelector) Invalidate() {}

// MoveUp moves selection up.
func (s *ModelSelector) MoveUp() {
	if s.selected > 0 {
		s.selected--
	}
}

// MoveDown moves selection down.
func (s *ModelSelector) MoveDown() {
	if s.selected < len(s.models)-1 {
		s.selected++
	}
}

// Confirm selects the current model.
func (s *ModelSelector) Confirm() {
	if s.selected < len(s.models) && s.onSelect != nil {
		s.onSelect(s.models[s.selected])
	}
}
