// ABOUTME: Tests for BashOutputModel: rendering, width truncation, exit codes
// ABOUTME: Verifies output lines are truncated to terminal width

package btea

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestBashOutputModel_Init(t *testing.T) {
	m := NewBashOutputModel("echo hello")
	if cmd := m.Init(); cmd != nil {
		t.Errorf("Init() returned non-nil cmd")
	}
}

func TestBashOutputModel_ViewShowsCommand(t *testing.T) {
	m := NewBashOutputModel("ls -la")
	m.width = 80
	m.AddOutput("file1\nfile2\n")
	view := m.View()
	if !strings.Contains(view, "ls -la") {
		t.Errorf("View() missing command; got %q", view)
	}
}

func TestBashOutputModel_ViewShowsExitCode(t *testing.T) {
	m := NewBashOutputModel("false")
	m.width = 80
	m.SetExitCode(1)
	view := m.View()
	if !strings.Contains(view, "exit code: 1") {
		t.Errorf("View() missing exit code; got %q", view)
	}
}

func TestBashOutputModel_ViewTruncatesLongLines(t *testing.T) {
	m := NewBashOutputModel("cat wide.txt")
	m.width = 40
	// Create a line much wider than 40 columns
	longLine := strings.Repeat("x", 100)
	m.AddOutput(longLine + "\n")

	view := m.View()
	lines := strings.SplitSeq(view, "\n")

	for line := range lines {
		if strings.Contains(line, "xxxx") {
			// This line should be truncated; check it ends with ellipsis
			if !strings.Contains(line, "…") {
				t.Errorf("long output line should be truncated with ellipsis; got %q", line)
			}
		}
	}
}

func TestBashOutputModel_WindowSizeMsg(t *testing.T) {
	m := NewBashOutputModel("ls")
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	bm := updated.(*BashOutputModel)
	if bm.width != 100 {
		t.Errorf("width = %d; want 100", bm.width)
	}
}

func TestBashOutputModel_ViewZeroExitCodeHidden(t *testing.T) {
	m := NewBashOutputModel("echo ok")
	m.width = 80
	m.AddOutput("ok\n")
	m.SetExitCode(0)
	view := m.View()
	if strings.Contains(view, "exit code") {
		t.Error("View() should not show exit code when it's 0")
	}
}
