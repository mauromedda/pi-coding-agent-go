// ABOUTME: Tests for AssistantMessage component: blank-line spacer and content rendering
// ABOUTME: Table-driven tests verify spacer, thinking indicator, and streaming text

package components

import (
	"strings"
	"testing"
	"time"

	"github.com/mauromedda/pi-coding-agent-go/pkg/tui"
	"github.com/mauromedda/pi-coding-agent-go/pkg/tui/width"
)

func TestAssistantMessage_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	msg := NewAssistantMessage()
	const iterations = 100
	done := make(chan struct{})

	// Writer goroutine: simulate agent streaming
	go func() {
		defer close(done)
		for i := 0; i < iterations; i++ {
			msg.AppendText("chunk ")
			msg.SetThinking("step")
			msg.Invalidate()
		}
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
			msg.Render(buf, 80)
		}
	}
}

func TestAssistantMessage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		text     string
		thinking string
		check    func(t *testing.T, lines []string)
	}{
		{
			name: "starts_with_blank_spacer_text_only",
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
			name:     "starts_with_blank_spacer_thinking",
			thinking: "reasoning",
			check: func(t *testing.T, lines []string) {
				t.Helper()
				if len(lines) < 2 {
					t.Fatalf("expected at least 2 lines (spacer + thinking), got %d", len(lines))
				}
				if lines[0] != "" {
					t.Errorf("first line should be blank spacer, got %q", lines[0])
				}
			},
		},
		{
			name: "renders_text_content",
			text: "response text",
			check: func(t *testing.T, lines []string) {
				t.Helper()
				if len(lines) < 2 {
					t.Fatalf("expected at least 2 lines, got %d", len(lines))
				}
				visible := width.StripANSI(lines[1])
				if !strings.Contains(visible, "response text") {
					t.Errorf("content line should contain text, got %q", visible)
				}
			},
		},
		{
			name:     "renders_thinking_indicator",
			thinking: "deep thought",
			check: func(t *testing.T, lines []string) {
				t.Helper()
				if len(lines) < 2 {
					t.Fatalf("expected at least 2 lines, got %d", len(lines))
				}
				visible := width.StripANSI(lines[1])
				if !strings.Contains(strings.ToLower(visible), "thinking") {
					t.Errorf("should show thinking indicator, got %q", visible)
				}
			},
		},
		{
			name: "empty_message_only_spacer",
			check: func(t *testing.T, lines []string) {
				t.Helper()
				// Empty message: should render spacer only
				if len(lines) != 1 {
					t.Errorf("empty message should render 1 line (spacer), got %d", len(lines))
				}
				if lines[0] != "" {
					t.Errorf("spacer line should be blank, got %q", lines[0])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			msg := NewAssistantMessage()
			if tt.text != "" {
				msg.AppendText(tt.text)
			}
			if tt.thinking != "" {
				msg.SetThinking(tt.thinking)
			}

			buf := tui.AcquireBuffer()
			defer tui.ReleaseBuffer(buf)

			msg.Render(buf, 80)
			tt.check(t, buf.Lines)
		})
	}
}

func TestAssistantMessage_RenderDoesNotBlockAppendText(t *testing.T) {
	t.Parallel()

	msg := NewAssistantMessage()

	// Pre-load ~100KB of text to make WrapTextWithAnsi expensive.
	bigChunk := strings.Repeat("word ", 20000)
	msg.AppendText(bigChunk)

	// Force an initial render so cache is primed, then dirty it.
	buf := tui.AcquireBuffer()
	msg.Render(buf, 80)
	tui.ReleaseBuffer(buf)
	msg.Invalidate()

	// Launch Render in a goroutine (it will re-wrap ~100KB).
	renderDone := make(chan struct{})
	go func() {
		defer close(renderDone)
		b := tui.AcquireBuffer()
		defer tui.ReleaseBuffer(b)
		msg.Render(b, 80)
	}()

	// Measure AppendText latency while Render is running.
	start := time.Now()
	msg.AppendText("extra")
	elapsed := time.Since(start)

	<-renderDone

	// AppendText must not be blocked by the expensive wrap; 1ms is generous.
	if elapsed > time.Millisecond {
		t.Errorf("AppendText took %v while Render was running; want < 1ms", elapsed)
	}
}

func TestAssistantMessage_EventualConsistency(t *testing.T) {
	t.Parallel()

	msg := NewAssistantMessage()
	msg.AppendText("initial")

	// First render: caches "initial".
	buf := tui.AcquireBuffer()
	msg.Render(buf, 80)
	tui.ReleaseBuffer(buf)

	// Append more text while no render is in progress.
	msg.AppendText(" extra")

	// Second render: must pick up the new text.
	buf2 := tui.AcquireBuffer()
	msg.Render(buf2, 80)

	found := false
	for _, line := range buf2.Lines {
		stripped := width.StripANSI(line)
		if strings.Contains(stripped, "extra") {
			found = true
			break
		}
	}
	tui.ReleaseBuffer(buf2)

	if !found {
		t.Error("second Render should include text appended after first Render")
	}
}

func TestAssistantMessage_thinking_has_spinner_char(t *testing.T) {
	t.Parallel()

	msg := NewAssistantMessage()
	msg.SetThinking("reasoning about code")

	buf := tui.AcquireBuffer()
	defer tui.ReleaseBuffer(buf)
	msg.Render(buf, 80)

	// The thinking indicator should contain a braille spinner character
	// instead of plain "thinking..."
	thinkingLine := ""
	for _, line := range buf.Lines {
		stripped := width.StripANSI(line)
		if strings.Contains(stripped, "hinking") || strings.Contains(stripped, "ndulating") {
			thinkingLine = stripped
			break
		}
	}
	if thinkingLine == "" {
		t.Fatal("expected a thinking indicator line")
	}
	// Should contain a spinner character (braille dots or similar)
	hasSpinner := false
	for _, r := range thinkingLine {
		if r >= '⠋' && r <= '⣿' { // braille range
			hasSpinner = true
			break
		}
	}
	if !hasSpinner {
		t.Errorf("thinking indicator should contain a braille spinner character, got %q", thinkingLine)
	}
}
