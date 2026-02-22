// ABOUTME: Tests for UserMessage component: blank-line spacer and visual distinction
// ABOUTME: Table-driven tests verify spacer, bold prefix, and content rendering

package components

import (
	"strings"
	"testing"

	"github.com/mauromedda/pi-coding-agent-go/pkg/tui"
	"github.com/mauromedda/pi-coding-agent-go/pkg/tui/width"
)

func TestUserMessage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		text  string
		check func(t *testing.T, lines []string)
	}{
		{
			name: "starts_with_blank_spacer",
			text: "hello",
			check: func(t *testing.T, lines []string) {
				t.Helper()
				if len(lines) < 2 {
					t.Fatalf("expected at least 2 lines (spacer + content), got %d", len(lines))
				}
				if lines[0] != "" {
					t.Errorf("first line should be blank spacer, got %q", lines[0])
				}
			},
		},
		{
			name: "renders_text_with_bold_prefix",
			text: "hello world",
			check: func(t *testing.T, lines []string) {
				t.Helper()
				if len(lines) < 2 {
					t.Fatalf("expected at least 2 lines, got %d", len(lines))
				}
				// Content is on line[1] (after spacer)
				visible := width.StripANSI(lines[1])
				if !strings.Contains(visible, "hello world") {
					t.Errorf("content line should contain text, got %q", visible)
				}
			},
		},
		{
			name: "has_bold_ansi",
			text: "test",
			check: func(t *testing.T, lines []string) {
				t.Helper()
				if len(lines) < 2 {
					t.Fatalf("expected at least 2 lines, got %d", len(lines))
				}
				// Bold ANSI should be present in content line
				if !strings.Contains(lines[1], "\x1b[1m") {
					t.Error("content line should contain bold ANSI code")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			msg := NewUserMessage(tt.text)
			buf := tui.AcquireBuffer()
			defer tui.ReleaseBuffer(buf)

			msg.Render(buf, 80)
			tt.check(t, buf.Lines)
		})
	}
}

func TestUserMessage_has_background_highlight(t *testing.T) {
	t.Parallel()

	msg := NewUserMessage("test prompt")
	buf := tui.AcquireBuffer()
	defer tui.ReleaseBuffer(buf)

	msg.Render(buf, 80)

	// The user message should have a background color ANSI code
	// to visually distinguish it from assistant messages.
	// Look for \x1b[48;5; (256-color bg) or \x1b[7m (reverse) or \x1b[100m (bright black bg)
	found := false
	for _, line := range buf.Lines {
		if strings.Contains(line, "\x1b[48;5;") || strings.Contains(line, "\x1b[7m") ||
			strings.Contains(line, "\x1b[100m") {
			found = true
			break
		}
	}
	if !found {
		t.Error("user message should have a background highlight ANSI code for visual distinction")
	}
}
