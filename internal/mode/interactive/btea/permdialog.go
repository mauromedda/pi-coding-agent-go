// ABOUTME: PermDialogModel is a Bubble Tea overlay for tool permission requests
// ABOUTME: Sends PermissionReply on channel; supports y/a/n/esc key bindings

package btea

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// DismissOverlayMsg signals that the current overlay should be removed.
type DismissOverlayMsg struct{}

// dismissOverlayCmd returns a DismissOverlayMsg.
func dismissOverlayCmd() tea.Msg {
	return DismissOverlayMsg{}
}

// PermDialogModel presents a permission approval dialog for a tool invocation.
// The user can allow (y), always-allow (a), or deny (n/esc).
// Implements tea.Model with value semantics.
type PermDialogModel struct {
	tool    string
	args    map[string]any
	replyCh chan<- PermissionReply
	width   int
}

// NewPermDialogModel creates a PermDialogModel for the given tool request.
func NewPermDialogModel(tool string, args map[string]any, replyCh chan<- PermissionReply) PermDialogModel {
	return PermDialogModel{
		tool:    tool,
		args:    args,
		replyCh: replyCh,
	}
}

// sendReply sends a PermissionReply on the reply channel without blocking.
// Uses a non-blocking select to prevent deadlock if the receiver has gone away.
func (m PermDialogModel) sendReply(reply PermissionReply) {
	select {
	case m.replyCh <- reply:
	default:
	}
}

// Init returns nil; no commands needed at startup.
func (m PermDialogModel) Init() tea.Cmd {
	return nil
}

// Update handles key messages for permission decisions.
func (m PermDialogModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		return m, nil
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyRunes:
			if len(msg.Runes) == 0 {
				break
			}
			switch msg.Runes[0] {
			case 'y':
				m.sendReply(PermissionReply{Allowed: true})
				return m, func() tea.Msg { return dismissOverlayCmd() }
			case 'a':
				m.sendReply(PermissionReply{Allowed: true, Always: true})
				return m, func() tea.Msg { return dismissOverlayCmd() }
			case 'n':
				m.sendReply(PermissionReply{Allowed: false})
				return m, func() tea.Msg { return dismissOverlayCmd() }
			}
		case tea.KeyEsc:
			m.sendReply(PermissionReply{Allowed: false})
			return m, func() tea.Msg { return dismissOverlayCmd() }
		}
	}
	return m, nil
}

// View renders the permission dialog as a bordered overlay box.
func (m PermDialogModel) View() string {
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

	boxWidth := 50
	innerWidth := boxWidth - 2
	contentWidth := boxWidth - 4
	border := bs.Render(vBorder)

	var b strings.Builder

	// Top border with title
	title := s.OverlayTitle.Render(" Permission Required ")
	titleLen := len(" Permission Required ")
	dashesLeft := max((innerWidth-titleLen)/2, 0)
	dashesRight := max(innerWidth-titleLen-dashesLeft, 0)
	b.WriteString(bs.Render(tl))
	b.WriteString(bs.Render(strings.Repeat(dash, dashesLeft)))
	b.WriteString(title)
	b.WriteString(bs.Render(strings.Repeat(dash, dashesRight)))
	b.WriteString(bs.Render(tr))
	b.WriteByte('\n')

	// Tool name
	writeBoxLine(&b, border, fmt.Sprintf("Tool: %s", s.Bold.Render(m.tool)), contentWidth)

	// Args
	if len(m.args) > 0 {
		writeBoxLine(&b, border, fmt.Sprintf("Args: %s", formatArgs(m.args)), contentWidth)
	}

	// Empty line
	writeBoxLine(&b, border, "", contentWidth)

	// Options
	allow := s.Success.Render("[y] Allow")
	always := s.Info.Render("[a] Always")
	deny := s.Error.Render("[n] Deny")
	writeBoxLine(&b, border, fmt.Sprintf("%s  %s  %s", allow, always, deny), contentWidth)

	// Bottom border
	b.WriteString(bs.Render(bl))
	b.WriteString(bs.Render(strings.Repeat(dash, innerWidth)))
	b.WriteString(bs.Render(br))

	return b.String()
}

// formatArgs formats a map as sorted key=value pairs.
func formatArgs(args map[string]any) string {
	keys := make([]string, 0, len(args))
	for k := range args {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%v", k, args[k]))
	}
	return strings.Join(parts, ", ")
}
