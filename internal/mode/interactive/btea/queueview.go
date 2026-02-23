// ABOUTME: QueueViewModel is a Bubble Tea overlay for viewing, editing, and reordering queued prompts
// ABOUTME: Vim-style navigation (j/k), delete (d), swap (J/K), edit (e/Enter), close (esc/q)

package btea

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mauromedda/pi-coding-agent-go/pkg/tui/width"
)

// QueueViewModel displays queued prompts with vim-style navigation and editing.
type QueueViewModel struct {
	items  []string
	cursor int
	width  int
}

// NewQueueViewModel creates a queue overlay with the given items.
func NewQueueViewModel(items []string, w int) QueueViewModel {
	cp := make([]string, len(items))
	copy(cp, items)
	return QueueViewModel{
		items: cp,
		width: w,
	}
}

// Init returns nil; no startup commands needed.
func (m QueueViewModel) Init() tea.Cmd { return nil }

// Update handles key events for queue management.
func (m QueueViewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)
	case tea.WindowSizeMsg:
		m.width = msg.Width
	}
	return m, nil
}

func (m QueueViewModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	// Navigation
	case "j", "down":
		if m.cursor < len(m.items)-1 {
			m.cursor++
		}
		return m, nil

	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
		}
		return m, nil

	// Delete
	case "d", "backspace":
		if len(m.items) == 0 {
			return m, nil
		}
		m.items = append(m.items[:m.cursor], m.items[m.cursor+1:]...)
		if len(m.items) == 0 {
			return m, m.closeCmd()
		}
		if m.cursor >= len(m.items) {
			m.cursor = len(m.items) - 1
		}
		return m, nil

	// Swap down (Shift+J)
	case "J":
		if m.cursor < len(m.items)-1 {
			m.items[m.cursor], m.items[m.cursor+1] = m.items[m.cursor+1], m.items[m.cursor]
			m.cursor++
		}
		return m, nil

	// Swap up (Shift+K)
	case "K":
		if m.cursor > 0 {
			m.items[m.cursor], m.items[m.cursor-1] = m.items[m.cursor-1], m.items[m.cursor]
			m.cursor--
		}
		return m, nil

	// Edit item: pop into editor
	case "e", "enter":
		if len(m.items) == 0 {
			return m, nil
		}
		text := m.items[m.cursor]
		idx := m.cursor
		return m, func() tea.Msg {
			return QueueEditMsg{Text: text, Index: idx}
		}

	// Close overlay
	case "esc", "q":
		return m, m.closeCmd()
	}

	return m, nil
}

func (m QueueViewModel) closeCmd() func() tea.Msg {
	items := make([]string, len(m.items))
	copy(items, m.items)
	return func() tea.Msg {
		return QueueUpdatedMsg{Items: items}
	}
}

// View renders the queue overlay.
func (m QueueViewModel) View() string {
	s := Styles()
	var b strings.Builder

	b.WriteString(s.Bold.Render("--- Queued Prompts ---"))
	b.WriteByte('\n')

	if len(m.items) == 0 {
		b.WriteString(s.Dim.Render("  (empty)"))
		b.WriteByte('\n')
	} else {
		maxW := m.width - 8 // account for prefix + padding
		if maxW < 10 {
			maxW = 10
		}
		for i, item := range m.items {
			prefix := "  "
			if i == m.cursor {
				prefix = "> "
			}
			display := item
			if width.VisibleWidth(display) > maxW {
				display = width.TruncateToWidth(display, maxW-3) + "..."
			}
			line := fmt.Sprintf("%s%d. %s", prefix, i+1, display)
			if i == m.cursor {
				b.WriteString(s.Selection.Render(line))
			} else {
				b.WriteString(s.Dim.Render(line))
			}
			b.WriteByte('\n')
		}
	}

	b.WriteString(s.Muted.Render("  j/k:nav  d:del  J/K:move  e:edit  esc:close"))
	return b.String()
}
