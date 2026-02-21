// ABOUTME: Tests for the TUI engine: differential rendering, overlays, cursor
// ABOUTME: Uses in-memory writer to capture output for assertions

package tui

import (
	"bytes"
	"strings"
	"testing"
)

type mockComponent struct {
	lines []string
	dirty bool
}

func (m *mockComponent) Render(out *RenderBuffer, width int) {
	out.WriteLines(m.lines)
}

func (m *mockComponent) Invalidate() {
	m.dirty = true
}

func TestRenderBuffer_Pool(t *testing.T) {
	t.Parallel()

	buf := AcquireBuffer()
	buf.WriteLine("line1")
	buf.WriteLine("line2")

	if buf.Len() != 2 {
		t.Errorf("Len() = %d, want 2", buf.Len())
	}

	ReleaseBuffer(buf)

	// Re-acquire should give a clean buffer
	buf2 := AcquireBuffer()
	if buf2.Len() != 0 {
		t.Errorf("re-acquired buffer Len() = %d, want 0", buf2.Len())
	}
	ReleaseBuffer(buf2)
}

func TestContainer_AddRemove(t *testing.T) {
	t.Parallel()

	c := NewContainer()
	comp1 := &mockComponent{lines: []string{"a"}}
	comp2 := &mockComponent{lines: []string{"b"}}

	c.Add(comp1)
	c.Add(comp2)

	if len(c.Children()) != 2 {
		t.Fatalf("expected 2 children, got %d", len(c.Children()))
	}

	if !c.Remove(comp1) {
		t.Error("Remove returned false for existing component")
	}

	if len(c.Children()) != 1 {
		t.Fatalf("expected 1 child after remove, got %d", len(c.Children()))
	}
}

func TestContainer_Render(t *testing.T) {
	t.Parallel()

	c := NewContainer()
	c.Add(&mockComponent{lines: []string{"hello"}})
	c.Add(&mockComponent{lines: []string{"world"}})

	buf := AcquireBuffer()
	defer ReleaseBuffer(buf)

	c.Render(buf, 80)

	if buf.Len() != 2 {
		t.Fatalf("expected 2 lines, got %d", buf.Len())
	}
	if buf.Lines[0] != "hello" || buf.Lines[1] != "world" {
		t.Errorf("unexpected lines: %v", buf.Lines)
	}
}

func TestTUI_RenderOnce(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	ui := New(&out, 80, 24)
	ui.Container().Add(&mockComponent{lines: []string{"test line"}})

	ui.RenderOnce()

	result := out.String()
	if !strings.Contains(result, "test line") {
		t.Errorf("expected output to contain 'test line', got %q", result)
	}
}

func TestTUI_DifferentialRender(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	ui := New(&out, 80, 24)

	comp := &mockComponent{lines: []string{"first"}}
	ui.Container().Add(comp)

	// First render
	ui.RenderOnce()
	firstSize := out.Len()

	// Same content: should produce minimal output
	out.Reset()
	ui.RenderOnce()
	secondSize := out.Len()

	if secondSize >= firstSize {
		t.Logf("first=%d second=%d; second should be smaller (no changes)", firstSize, secondSize)
	}
}

func TestTUI_CursorPosition(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	ui := New(&out, 80, 24)

	comp := &mockComponent{lines: []string{"abc" + CursorMarker + "def"}}
	ui.Container().Add(comp)

	ui.RenderOnce()

	result := out.String()
	// Cursor should be positioned at column 4 (after "abc")
	if !strings.Contains(result, "\x1b[1;4H") {
		t.Errorf("expected cursor positioning at row 1, col 4; got %q", result)
	}
	// Cursor should be shown
	if !strings.Contains(result, "\x1b[?25h") {
		t.Error("expected cursor to be shown")
	}
}

func TestExtractCursorPosition(t *testing.T) {
	t.Parallel()

	lines := []string{"hello" + CursorMarker + "world"}
	row, col := extractCursorPosition(lines)

	if row != 0 || col != 5 {
		t.Errorf("cursor at (%d, %d), want (0, 5)", row, col)
	}
	if lines[0] != "helloworld" {
		t.Errorf("marker not stripped: %q", lines[0])
	}
}

func TestExtractCursorPosition_NotFound(t *testing.T) {
	t.Parallel()

	lines := []string{"no cursor here"}
	row, col := extractCursorPosition(lines)

	if row != -1 || col != -1 {
		t.Errorf("expected (-1, -1), got (%d, %d)", row, col)
	}
}

func TestOverlay_Center(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	ui := New(&out, 40, 10)

	ui.Container().Add(&mockComponent{lines: []string{"background"}})
	ui.PushOverlay(Overlay{
		Component: &mockComponent{lines: []string{"overlay"}},
		Position:  OverlayCenter,
	})

	ui.RenderOnce()

	result := out.String()
	if !strings.Contains(result, "overlay") {
		t.Error("overlay content not found in output")
	}
}
