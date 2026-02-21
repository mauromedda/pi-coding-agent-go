// ABOUTME: Tests for basic TUI components: Text, Box, Spacer, Loader, TruncatedText
// ABOUTME: Verifies rendering output and state management

package component

import (
	"testing"

	"github.com/mauromedda/pi-coding-agent-go/pkg/tui"
)

func TestText_Render(t *testing.T) {
	t.Parallel()

	comp := NewText("hello\nworld")
	buf := tui.AcquireBuffer()
	defer tui.ReleaseBuffer(buf)

	comp.Render(buf, 80)

	if buf.Len() != 2 {
		t.Fatalf("expected 2 lines, got %d", buf.Len())
	}
	if buf.Lines[0] != "hello" || buf.Lines[1] != "world" {
		t.Errorf("unexpected lines: %v", buf.Lines)
	}
}

func TestText_SetContent(t *testing.T) {
	t.Parallel()

	comp := NewText("old")
	comp.SetContent("new")

	buf := tui.AcquireBuffer()
	defer tui.ReleaseBuffer(buf)

	comp.Render(buf, 80)

	if buf.Len() != 1 || buf.Lines[0] != "new" {
		t.Errorf("expected 'new', got %v", buf.Lines)
	}
}

func TestSpacer_Render(t *testing.T) {
	t.Parallel()

	sp := NewSpacer(3)
	buf := tui.AcquireBuffer()
	defer tui.ReleaseBuffer(buf)

	sp.Render(buf, 80)

	if buf.Len() != 3 {
		t.Errorf("expected 3 lines, got %d", buf.Len())
	}
}

func TestLoader_Tick(t *testing.T) {
	t.Parallel()

	l := NewLoader("loading")
	buf := tui.AcquireBuffer()
	defer tui.ReleaseBuffer(buf)

	l.Render(buf, 80)
	first := buf.Lines[0]

	l.Tick()
	buf.Reset()
	l.Render(buf, 80)
	second := buf.Lines[0]

	if first == second {
		t.Error("expected different frames after Tick")
	}
}

func TestBox_Render(t *testing.T) {
	t.Parallel()

	child := NewText("content")
	box := NewBox(child).WithPadding(1)

	buf := tui.AcquireBuffer()
	defer tui.ReleaseBuffer(buf)

	box.Render(buf, 40)

	// Should have: 1 top pad + 1 content + 1 bottom pad = 3 lines
	if buf.Len() != 3 {
		t.Fatalf("expected 3 lines, got %d: %v", buf.Len(), buf.Lines)
	}
}

func TestTruncatedText_Fits(t *testing.T) {
	t.Parallel()

	tt := NewTruncatedText("short")
	buf := tui.AcquireBuffer()
	defer tui.ReleaseBuffer(buf)

	tt.Render(buf, 80)

	if buf.Len() != 1 || buf.Lines[0] != "short" {
		t.Errorf("expected 'short', got %v", buf.Lines)
	}
}
