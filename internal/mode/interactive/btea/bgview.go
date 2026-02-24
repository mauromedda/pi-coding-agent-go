// ABOUTME: BackgroundViewModel is a Bubble Tea overlay listing background tasks
// ABOUTME: Navigate (j/k), review completed (Enter), dismiss (d), cancel running (c), close (Esc)

package btea

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mauromedda/pi-coding-agent-go/pkg/tui/width"
)

// BackgroundViewModel displays background tasks as a centered overlay.
type BackgroundViewModel struct {
	tasks  []BackgroundTask
	cursor int
	width  int
	height int
}

// NewBackgroundViewModel creates the overlay from a snapshot of tasks.
func NewBackgroundViewModel(tasks []BackgroundTask, w, h int) BackgroundViewModel {
	cp := make([]BackgroundTask, len(tasks))
	copy(cp, tasks)
	return BackgroundViewModel{
		tasks:  cp,
		width:  w,
		height: h,
	}
}

// Init returns nil; no startup commands needed.
func (m BackgroundViewModel) Init() tea.Cmd { return nil }

// Update handles key events for background task management.
func (m BackgroundViewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}
	return m, nil
}

func (m BackgroundViewModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		if m.cursor < len(m.tasks)-1 {
			m.cursor++
		}
		return m, nil

	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
		}
		return m, nil

	case "enter":
		if len(m.tasks) == 0 {
			return m, nil
		}
		task := m.tasks[m.cursor]
		if task.Status == BGDone || task.Status == BGFailed {
			id := task.ID
			return m, func() tea.Msg { return BackgroundTaskReviewMsg{TaskID: id} }
		}
		return m, nil

	case "d":
		if len(m.tasks) == 0 {
			return m, nil
		}
		task := m.tasks[m.cursor]
		if task.Status == BGDone || task.Status == BGFailed {
			id := task.ID
			// Remove from local view and send remove message to sync BackgroundManager.
			m.tasks = append(m.tasks[:m.cursor], m.tasks[m.cursor+1:]...)
			if m.cursor >= len(m.tasks) && m.cursor > 0 {
				m.cursor = len(m.tasks) - 1
			}
			if len(m.tasks) == 0 {
				return m, func() tea.Msg { return BackgroundTaskRemoveMsg{TaskID: id} }
			}
			return m, func() tea.Msg { return BackgroundTaskRemoveMsg{TaskID: id} }
		}
		return m, nil

	case "c":
		if len(m.tasks) == 0 {
			return m, nil
		}
		task := m.tasks[m.cursor]
		if task.Status == BGRunning {
			id := task.ID
			return m, func() tea.Msg { return BackgroundTaskCancelMsg{TaskID: id} }
		}
		return m, nil

	case "esc", "q":
		return m, func() tea.Msg { return DismissOverlayMsg{} }
	}
	return m, nil
}

// View renders the background tasks overlay as a bordered box.
func (m BackgroundViewModel) View() string {
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

	boxWidth := max(m.width*2/5, 44)
	if boxWidth > m.width-4 {
		boxWidth = max(m.width-4, 44)
	}
	innerWidth := max(boxWidth-2, 0)
	contentWidth := max(boxWidth-4, 20)
	border := bs.Render(vBorder)

	var b strings.Builder

	// Top border with title
	title := s.OverlayTitle.Render(" Background Tasks ")
	titleLen := len(" Background Tasks ")
	dashesLeft := max((innerWidth-titleLen)/2, 0)
	dashesRight := max(innerWidth-titleLen-dashesLeft, 0)
	b.WriteString(bs.Render(tl))
	b.WriteString(bs.Render(strings.Repeat(dash, dashesLeft)))
	b.WriteString(title)
	b.WriteString(bs.Render(strings.Repeat(dash, dashesRight)))
	b.WriteString(bs.Render(tr))
	b.WriteByte('\n')

	if len(m.tasks) == 0 {
		writeBoxLine(&b, border, s.Dim.Render("(no background tasks)"), contentWidth)
	} else {
		maxW := contentWidth - 12 // status icon + ID prefix
		if maxW < 10 {
			maxW = 10
		}
		for i, task := range m.tasks {
			prefix := "  "
			if i == m.cursor {
				prefix = "> "
			}

			icon := statusIcon(task.Status)
			prompt := task.Prompt
			if width.VisibleWidth(prompt) > maxW {
				prompt = width.TruncateToWidth(prompt, maxW-3) + "..."
			}

			line := fmt.Sprintf("%s%s [%s] %s", prefix, icon, task.ID, prompt)
			if i == m.cursor {
				writeBoxLine(&b, border, s.Selection.Render(line), contentWidth)
			} else {
				writeBoxLine(&b, border, s.Dim.Render(line), contentWidth)
			}
		}
	}

	// Hint line
	writeBoxLine(&b, border, s.Muted.Render("j/k:nav  enter:review  d:dismiss  c:cancel  esc:close"), contentWidth)

	// Bottom border
	b.WriteString(bs.Render(bl))
	b.WriteString(bs.Render(strings.Repeat(dash, innerWidth)))
	b.WriteString(bs.Render(br))

	return b.String()
}

func statusIcon(s BackgroundStatus) string {
	switch s {
	case BGRunning:
		return "⠋"
	case BGDone:
		return "✓"
	case BGFailed:
		return "✗"
	default:
		return "?"
	}
}
