// ABOUTME: Tests for ToolExec component: blank-line spacer and status rendering
// ABOUTME: Table-driven tests verify spacer, running/done/error status indicators

package components

import (
	"strings"
	"testing"

	"github.com/mauromedda/pi-coding-agent-go/pkg/tui"
	"github.com/mauromedda/pi-coding-agent-go/pkg/tui/width"
)

func TestToolExec_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	te := NewToolExec("bash", `{"command":"ls"}`)
	const iterations = 100
	done := make(chan struct{})

	// Writer goroutine: simulate agent streaming output
	go func() {
		defer close(done)
		for i := 0; i < iterations; i++ {
			te.AppendOutput("line\n")
		}
		te.SetDone("")
	}()

	// Reader goroutine (this one): simulate render loop
	buf := tui.AcquireBuffer()
	defer tui.ReleaseBuffer(buf)
	for {
		select {
		case <-done:
			return
		default:
			buf.Lines = buf.Lines[:0]
			te.Render(buf, 80)
		}
	}
}

func TestToolExec(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		tool    string
		args    string
		done    bool
		errMsg  string
		check   func(t *testing.T, lines []string)
	}{
		{
			name: "starts_with_blank_spacer",
			tool: "bash",
			args: `{"command":"ls"}`,
			check: func(t *testing.T, lines []string) {
				t.Helper()
				if len(lines) < 2 {
					t.Fatalf("expected at least 2 lines (spacer + status), got %d", len(lines))
				}
				if lines[0] != "" {
					t.Errorf("first line should be blank spacer, got %q", lines[0])
				}
			},
		},
		{
			name: "shows_tool_name_in_content",
			tool: "read_file",
			args: `{"path":"/tmp/x"}`,
			check: func(t *testing.T, lines []string) {
				t.Helper()
				if len(lines) < 2 {
					t.Fatalf("expected at least 2 lines, got %d", len(lines))
				}
				visible := width.StripANSI(lines[1])
				if !strings.Contains(visible, "read_file") {
					t.Errorf("content line should contain tool name, got %q", visible)
				}
			},
		},
		{
			name: "running_shows_spinner",
			tool: "bash",
			args: "",
			check: func(t *testing.T, lines []string) {
				t.Helper()
				if len(lines) < 2 {
					t.Fatalf("expected at least 2 lines, got %d", len(lines))
				}
				// Yellow spinner should be present
				if !strings.Contains(lines[1], "\x1b[33m") {
					t.Error("running tool should have yellow ANSI code")
				}
			},
		},
		{
			name: "done_shows_green_check",
			tool: "bash",
			args: "",
			done: true,
			check: func(t *testing.T, lines []string) {
				t.Helper()
				if len(lines) < 2 {
					t.Fatalf("expected at least 2 lines, got %d", len(lines))
				}
				if !strings.Contains(lines[1], "\x1b[32m") {
					t.Error("completed tool should have green ANSI code")
				}
			},
		},
		{
			name:   "error_shows_red_cross_and_message",
			tool:   "bash",
			args:   "",
			done:   true,
			errMsg: "command failed",
			check: func(t *testing.T, lines []string) {
				t.Helper()
				if len(lines) < 3 {
					t.Fatalf("expected at least 3 lines (spacer + status + error), got %d", len(lines))
				}
				if !strings.Contains(lines[1], "\x1b[31m") {
					t.Error("error tool should have red ANSI code")
				}
				visible := width.StripANSI(lines[2])
				if !strings.Contains(visible, "command failed") {
					t.Errorf("error line should contain error message, got %q", visible)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			te := NewToolExec(tt.tool, tt.args)
			if tt.done {
				te.SetDone(tt.errMsg)
			}

			buf := tui.AcquireBuffer()
			defer tui.ReleaseBuffer(buf)

			te.Render(buf, 80)
			tt.check(t, buf.Lines)
		})
	}
}
