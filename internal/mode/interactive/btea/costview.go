// ABOUTME: Token and cost dashboard overlay toggled with ctrl+t
// ABOUTME: Shows session totals, per-call breakdown, and budget status

package btea

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// CostViewModel displays token usage and cost information.
type CostViewModel struct {
	totalInput    int
	totalOutput   int
	totalCost     float64
	callCount     int
	budgetUSD     float64
	budgetUsedPct float64
	width         int
	height        int
}

// NewCostViewModel creates a cost dashboard overlay.
func NewCostViewModel(input, output, calls int, cost, budget, budgetPct float64) CostViewModel {
	return CostViewModel{
		totalInput:    input,
		totalOutput:   output,
		callCount:     calls,
		totalCost:     cost,
		budgetUSD:     budget,
		budgetUsedPct: budgetPct,
	}
}

// Init returns nil; no startup commands needed.
func (m CostViewModel) Init() tea.Cmd { return nil }

// Update handles key events for the cost dashboard.
func (m CostViewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "ctrl+t", "q":
			return m, func() tea.Msg { return DismissOverlayMsg{} }
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}
	return m, nil
}

// View renders the cost dashboard as a bordered overlay box.
func (m CostViewModel) View() string {
	s := Styles()
	bs := s.OverlayBorder

	const (
		dash    = "─"
		vBorder = "│"
		tl      = "╭"
		tr      = "╮"
		bl      = "╰"
		br      = "╯"
	)

	boxWidth := 44 // fixed width for cost dashboard
	innerWidth := boxWidth - 2
	contentWidth := boxWidth - 4
	border := bs.Render(vBorder)

	var b strings.Builder

	// Top border with title
	title := s.OverlayTitle.Render(" Token & Cost Dashboard ")
	titleLen := len(" Token & Cost Dashboard ")
	dashesLeft := max((innerWidth-titleLen)/2, 0)
	dashesRight := max(innerWidth-titleLen-dashesLeft, 0)
	b.WriteString(bs.Render(tl))
	b.WriteString(bs.Render(strings.Repeat(dash, dashesLeft)))
	b.WriteString(title)
	b.WriteString(bs.Render(strings.Repeat(dash, dashesRight)))
	b.WriteString(bs.Render(tr))
	b.WriteByte('\n')

	// Content lines
	writeBoxLine(&b, border, fmt.Sprintf("Input tokens:  %s", formatNumber(m.totalInput)), contentWidth)
	writeBoxLine(&b, border, fmt.Sprintf("Output tokens: %s", formatNumber(m.totalOutput)), contentWidth)
	writeBoxLine(&b, border, fmt.Sprintf("Total cost:    $%.2f", m.totalCost), contentWidth)
	writeBoxLine(&b, border, fmt.Sprintf("API calls:     %d", m.callCount), contentWidth)
	writeBoxLine(&b, border, fmt.Sprintf("Budget:        $%.2f", m.budgetUSD), contentWidth)
	writeBoxLine(&b, border, fmt.Sprintf("Used:          %.1f%%", m.budgetUsedPct), contentWidth)

	// Hint line
	writeBoxLine(&b, border, s.Dim.Render("Press ESC to close"), contentWidth)

	// Bottom border
	b.WriteString(bs.Render(bl))
	b.WriteString(bs.Render(strings.Repeat(dash, innerWidth)))
	b.WriteString(bs.Render(br))

	return b.String()
}

// formatNumber formats an integer with thousand separators.
func formatNumber(n int) string {
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}

	var parts []string
	for len(s) > 3 {
		parts = append([]string{s[len(s)-3:]}, parts...)
		s = s[:len(s)-3]
	}
	if len(s) > 0 {
		parts = append([]string{s}, parts...)
	}
	return strings.Join(parts, ",")
}
