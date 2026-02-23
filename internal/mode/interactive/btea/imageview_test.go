// ABOUTME: Tests for the Bubble Tea image view model
// ABOUTME: Verifies eager rendering, placeholder behavior, and value-type safety

package btea

import (
	"bytes"
	"image"
	"image/png"
	"os"
	"strings"
	"testing"
)

func makeTestPNG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestImageViewModel_View(t *testing.T) {
	// Force no image protocol for predictable half-block output
	os.Unsetenv("KITTY_WINDOW_ID")
	os.Unsetenv("ITERM_SESSION_ID")
	os.Unsetenv("TERM_PROGRAM")

	data := makeTestPNG(t, 20, 10)
	m := NewImageViewModel(data, "image/png", 40)

	view := m.View()
	if view == "" {
		t.Error("expected non-empty view")
	}
	if !strings.Contains(view, "▄") && !strings.Contains(view, "[Image:") {
		t.Errorf("expected half-block or placeholder, got: %q", view)
	}
}

func TestImageViewModel_InvalidData(t *testing.T) {
	m := NewImageViewModel([]byte("not an image"), "image/png", 40)
	view := m.View()
	if !strings.Contains(view, "[Image:") {
		t.Errorf("expected placeholder for invalid data, got: %q", view)
	}
}

func TestImageViewModel_EmptyData(t *testing.T) {
	m := NewImageViewModel(nil, "image/png", 40)
	view := m.View()
	if view != "" {
		t.Errorf("expected empty view for nil data, got: %q", view)
	}
}

func TestImageViewModel_ValueTypeSafety(t *testing.T) {
	// Verify that View() works correctly after value copy (no pointer receiver issues)
	os.Unsetenv("KITTY_WINDOW_ID")
	os.Unsetenv("ITERM_SESSION_ID")
	os.Unsetenv("TERM_PROGRAM")

	data := makeTestPNG(t, 20, 10)
	m := NewImageViewModel(data, "image/png", 40)

	// Copy the model (simulates Bubble Tea's value semantics)
	m2 := m

	view1 := m.View()
	view2 := m2.View()
	if view1 != view2 {
		t.Error("expected identical output from original and copy")
	}
	if view1 == "" {
		t.Error("expected non-empty output")
	}
}

func TestImageViewModel_ZeroWidth(t *testing.T) {
	data := makeTestPNG(t, 20, 10)
	m := NewImageViewModel(data, "image/png", 0)
	if m.View() != "" {
		t.Error("expected empty view for zero width")
	}
}
