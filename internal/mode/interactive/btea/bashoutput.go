// ABOUTME: BashOutputModel is a Bubble Tea leaf that renders bash command output
// ABOUTME: Special handling for bash commands executed via !

package btea

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// BashOutputModel renders a bash command with its output.
type BashOutputModel struct {
	command  string
	output   strings.Builder
	exitCode int
	width    int
}

// NewBashOutputModel creates a new bash output model.
func NewBashOutputModel(command string) *BashOutputModel {
	return &BashOutputModel{
		command: command,
	}
}

// Init returns nil; no commands needed for a leaf model.
func (m *BashOutputModel) Init() tea.Cmd {
	return nil
}

// Update handles window size changes.
func (m *BashOutputModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
	}
	return m, nil
}

// AddOutput appends output to the model.
func (m *BashOutputModel) AddOutput(text string) {
	m.output.WriteString(text)
}

// SetCommand sets the command being executed.
func (m *BashOutputModel) SetCommand(command string) {
	m.command = command
}

// SetExitCode sets the exit code for the command.
func (m *BashOutputModel) SetExitCode(code int) {
	m.exitCode = code
}

// View renders the bash command output with proper alignment and styling.
func (m *BashOutputModel) View() string {
	s := Styles()
	var b strings.Builder

	// Command line - show command in the command's color (Bash tool color)
	b.WriteString("\n")
	cmdLine := fmt.Sprintf("%s %s", s.ToolBash.Render("!"), s.ToolBash.Render(m.command))
	b.WriteString(cmdLine + "\n")

	// Output: pass through raw lines to preserve ANSI from commands
	raw := m.output.String()
	if raw != "" {
		lines := strings.Split(strings.TrimRight(raw, "\n"), "\n")
		for _, line := range lines {
			b.WriteString(line + "\n")
		}
	}

	// Exit code (only when non-zero)
	if m.exitCode != 0 {
		b.WriteString(s.Error.Render(fmt.Sprintf("exit code: %d", m.exitCode)) + "\n")
	}

	return b.String()
}
