// ABOUTME: Tests for the high-level image rendering dispatcher
// ABOUTME: Verifies protocol routing, fallback to half-block, and error handling

package image

import (
	"os"
	"strings"
	"testing"
)

func TestRender_HalfBlockFallback(t *testing.T) {
	withCleanEnv(t, func() {
		// Force ProtoNone (no image protocol)
		data := makePNG(t, 40, 20)
		lines, err := Render(data, "image/png", 40)
		if err != nil {
			t.Fatal(err)
		}
		if len(lines) == 0 {
			t.Error("expected non-empty output")
		}
		// Half-block output should contain ▄ characters
		for _, line := range lines {
			if !strings.Contains(line, "▄") {
				t.Errorf("expected half-block chars in fallback, got: %q", line)
			}
		}
	})
}

func TestRender_KittyProtocol(t *testing.T) {
	withCleanEnv(t, func() {
		os.Setenv("KITTY_WINDOW_ID", "1")
		data := makePNG(t, 40, 20)
		lines, err := Render(data, "image/png", 80)
		if err != nil {
			t.Fatal(err)
		}
		if len(lines) != 1 {
			t.Fatalf("expected 1 line for Kitty output, got %d", len(lines))
		}
		if !strings.Contains(lines[0], "\x1b_G") {
			t.Error("expected Kitty APC escape in output")
		}
	})
}

func TestRender_ITerm2Protocol(t *testing.T) {
	withCleanEnv(t, func() {
		os.Setenv("ITERM_SESSION_ID", "w0t0p0:12345")
		data := makePNG(t, 40, 20)
		lines, err := Render(data, "image/png", 80)
		if err != nil {
			t.Fatal(err)
		}
		if len(lines) != 1 {
			t.Fatalf("expected 1 line for iTerm2 output, got %d", len(lines))
		}
		if !strings.Contains(lines[0], "\x1b]1337;File=") {
			t.Error("expected iTerm2 OSC 1337 escape in output")
		}
	})
}

func TestRender_EmptyData(t *testing.T) {
	_, err := Render(nil, "image/png", 80)
	if err == nil {
		t.Error("expected error for nil data")
	}
}

func TestRender_InvalidData(t *testing.T) {
	withCleanEnv(t, func() {
		// ProtoNone: should fall back to placeholder on decode error
		lines, err := Render([]byte("not an image"), "image/png", 80)
		if err != nil {
			t.Fatal(err)
		}
		if len(lines) != 1 {
			t.Fatalf("expected 1 placeholder line, got %d", len(lines))
		}
		if !strings.Contains(lines[0], "[Image:") {
			t.Errorf("expected placeholder text, got: %q", lines[0])
		}
	})
}

func TestRender_JPEGInput(t *testing.T) {
	withCleanEnv(t, func() {
		// ProtoNone: JPEG should render as half-block
		data := makeJPEG(t, 40, 20)
		lines, err := Render(data, "image/jpeg", 40)
		if err != nil {
			t.Fatal(err)
		}
		if len(lines) == 0 {
			t.Error("expected non-empty output for JPEG")
		}
	})
}
