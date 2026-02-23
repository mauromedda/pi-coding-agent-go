// ABOUTME: Plan review overlay for approving or rejecting generated plans
// ABOUTME: Shows structured plan content with approve/edit/reject key bindings

package btea

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// PlanApprovedMsg signals the user approved the plan.
type PlanApprovedMsg struct{}

// PlanRejectedMsg signals the user rejected the plan.
type PlanRejectedMsg struct{}

// PlanViewModel displays a generated plan for user review.
type PlanViewModel struct {
	plan   string
	width  int
	height int
	scroll int // scroll offset for long plans
}

// NewPlanViewModel creates a plan review overlay.
func NewPlanViewModel(plan string) PlanViewModel {
	return PlanViewModel{
		plan: plan,
	}
}

// Init returns nil; no startup commands needed.
func (m PlanViewModel) Init() tea.Cmd { return nil }

// Update handles key events for plan review.
func (m PlanViewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "y", "enter":
			return m, func() tea.Msg { return PlanApprovedMsg{} }
		case "n", "esc":
			return m, func() tea.Msg { return PlanRejectedMsg{} }
		case "j", "down":
			m.scroll++
		case "k", "up":
			if m.scroll > 0 {
				m.scroll--
			}
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}
	return m, nil
}

// View renders the plan with a header and scroll support.
func (m PlanViewModel) View() string {
	header := "Plan Review (y=approve, n=reject, j/k=scroll)"

	lines := strings.Split(m.plan, "\n")

	// Apply scroll offset
	start := m.scroll
	if start > len(lines) {
		start = len(lines)
	}
	visible := lines[start:]

	// Limit visible lines to available height (reserve 3 for header + footer + border)
	maxVisible := len(visible)
	if m.height > 4 && maxVisible > m.height-4 {
		maxVisible = m.height - 4
	}
	if maxVisible > 0 {
		visible = visible[:maxVisible]
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("--- %s ---\n", header))
	b.WriteString(strings.Join(visible, "\n"))
	b.WriteString("\n---")

	return b.String()
}
