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

// View renders the plan as a bordered overlay box with scroll support.
func (m PlanViewModel) View() string {
	s := Styles()
	bs := s.OverlayBorder

	// Box geometry: 60% of terminal width, clamped
	boxWidth := max(m.width*3/5, 40)
	if boxWidth > m.width-4 {
		boxWidth = max(m.width-4, 40)
	}
	contentWidth := max(boxWidth-4, 20) // │ + space + content + space + │

	lines := strings.Split(m.plan, "\n")
	totalLines := len(lines)

	// Apply scroll offset
	start := min(m.scroll, totalLines)
	visible := lines[start:]

	// Reserve rows for: top border(1) + header(1) + separator(1) + footer(1) + bottom border(1) = 5
	maxVisible := len(visible)
	boxHeight := max(m.height*3/5, 10)
	contentRows := max(boxHeight-5, 3)
	if maxVisible > contentRows {
		maxVisible = contentRows
	}
	if maxVisible > 0 {
		visible = visible[:maxVisible]
	}
	end := start + maxVisible

	// Border chars
	const (
		dash    = "─"
		vBorder = "│"
		tl      = "╭"
		tr      = "╮"
		bl      = "╰"
		br      = "╯"
	)

	border := bs.Render(vBorder)
	innerWidth := max(boxWidth-2, 0)

	var b strings.Builder

	// Top border with title
	title := s.OverlayTitle.Render(" Plan Review ")
	titleLen := len(" Plan Review ") // visible chars
	dashesLeft := max((innerWidth-titleLen)/2, 0)
	dashesRight := max(innerWidth-titleLen-dashesLeft, 0)
	b.WriteString(bs.Render(tl))
	b.WriteString(bs.Render(strings.Repeat(dash, dashesLeft)))
	b.WriteString(title)
	b.WriteString(bs.Render(strings.Repeat(dash, dashesRight)))
	b.WriteString(bs.Render(tr))
	b.WriteByte('\n')

	// Keybinding hints line
	hints := s.Dim.Render("y=approve  n=reject  j/k=scroll")
	writeBoxLine(&b, border, hints, contentWidth)

	// Separator
	writeBoxLine(&b, border, bs.Render(strings.Repeat(dash, contentWidth)), contentWidth)

	// Plan content lines
	for _, line := range visible {
		// Truncate long lines to fit content area
		writeBoxLine(&b, border, line, contentWidth)
	}

	// Scroll indicator
	if totalLines > contentRows {
		indicator := s.Dim.Render(fmt.Sprintf("lines %d-%d of %d", start+1, end, totalLines))
		writeBoxLine(&b, border, indicator, contentWidth)
	}

	// Bottom border
	b.WriteString(bs.Render(bl))
	b.WriteString(bs.Render(strings.Repeat(dash, innerWidth)))
	b.WriteString(bs.Render(br))

	return b.String()
}
