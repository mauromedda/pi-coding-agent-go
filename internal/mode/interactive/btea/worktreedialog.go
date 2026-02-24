// ABOUTME: WorktreeDialogModel is a Bubble Tea overlay for session worktree exit actions
// ABOUTME: Presents Merge/Keep/Discard options; sends WorktreeExitMsg on selection

package btea

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// WorktreeExitAction identifies the user's choice for worktree cleanup.
type WorktreeExitAction int

const (
	// WorktreeActionMerge merges the worktree branch into the original branch.
	WorktreeActionMerge WorktreeExitAction = iota
	// WorktreeActionKeep leaves the worktree and branch in place.
	WorktreeActionKeep
	// WorktreeActionDiscard removes the worktree and deletes the branch.
	WorktreeActionDiscard
)

// WorktreeExitMsg carries the user's worktree exit choice.
type WorktreeExitMsg struct {
	Action WorktreeExitAction
}

// WorktreeDialogModel renders a centered dialog for worktree cleanup on exit.
type WorktreeDialogModel struct {
	branch string
	width  int
}

// NewWorktreeDialogModel creates the exit dialog for the given worktree branch.
func NewWorktreeDialogModel(branch string, w int) WorktreeDialogModel {
	return WorktreeDialogModel{branch: branch, width: w}
}

// Init returns nil; no startup commands needed.
func (m WorktreeDialogModel) Init() tea.Cmd { return nil }

// Update handles key events for worktree exit actions.
func (m WorktreeDialogModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)
	case tea.WindowSizeMsg:
		m.width = msg.Width
	}
	return m, nil
}

func (m WorktreeDialogModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "m":
		return m, func() tea.Msg { return WorktreeExitMsg{Action: WorktreeActionMerge} }
	case "k":
		return m, func() tea.Msg { return WorktreeExitMsg{Action: WorktreeActionKeep} }
	case "d":
		return m, func() tea.Msg { return WorktreeExitMsg{Action: WorktreeActionDiscard} }
	case "esc":
		return m, func() tea.Msg { return DismissOverlayMsg{} }
	}
	return m, nil
}

// View renders the worktree exit dialog.
func (m WorktreeDialogModel) View() string {
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

	boxWidth := 48
	if boxWidth > m.width-4 {
		boxWidth = max(m.width-4, 40)
	}
	innerWidth := max(boxWidth-2, 0)
	contentWidth := max(boxWidth-4, 20)
	border := bs.Render(vBorder)

	var b strings.Builder

	// Top border with title
	title := s.OverlayTitle.Render(" Session Worktree ")
	titleLen := len(" Session Worktree ")
	dashesLeft := max((innerWidth-titleLen)/2, 0)
	dashesRight := max(innerWidth-titleLen-dashesLeft, 0)
	b.WriteString(bs.Render(tl))
	b.WriteString(bs.Render(strings.Repeat(dash, dashesLeft)))
	b.WriteString(title)
	b.WriteString(bs.Render(strings.Repeat(dash, dashesRight)))
	b.WriteString(bs.Render(tr))
	b.WriteByte('\n')

	// Branch name
	writeBoxLine(&b, border, s.Dim.Render("Branch: "+m.branch), contentWidth)

	// Empty line
	writeBoxLine(&b, border, "", contentWidth)

	// Options
	writeBoxLine(&b, border, s.Success.Render("[m]")+" Merge into original branch", contentWidth)
	writeBoxLine(&b, border, s.Info.Render("[k]")+" Keep worktree (no merge)", contentWidth)
	writeBoxLine(&b, border, s.Error.Render("[d]")+" Discard worktree and branch", contentWidth)

	// Empty line
	writeBoxLine(&b, border, "", contentWidth)

	// Hint
	writeBoxLine(&b, border, s.Muted.Render("esc: cancel exit"), contentWidth)

	// Bottom border
	b.WriteString(bs.Render(bl))
	b.WriteString(bs.Render(strings.Repeat(dash, innerWidth)))
	b.WriteString(bs.Render(br))

	return b.String()
}
