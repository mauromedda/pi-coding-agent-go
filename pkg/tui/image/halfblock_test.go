// ABOUTME: Tests for ANSI half-block character image renderer
// ABOUTME: Verifies line count, ANSI escape sequences, and reset codes

package image

import (
	"image"
	"image/color"
	"strings"
	"testing"
)

func TestRenderHalfBlock_BasicOutput(t *testing.T) {
	// 4x4 red image → 2 lines (2 pixel rows per line)
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for y := range 4 {
		for x := range 4 {
			img.Set(x, y, color.RGBA{R: 255, A: 255})
		}
	}

	lines := RenderHalfBlock(img, 4)
	if len(lines) != 2 {
		t.Errorf("expected 2 lines for 4px height, got %d", len(lines))
	}

	for i, line := range lines {
		// Each line should contain the lower half block character
		if !strings.Contains(line, "▄") {
			t.Errorf("line %d missing ▄ character", i)
		}
		// Each line should end with reset
		if !strings.HasSuffix(line, "\x1b[0m") {
			t.Errorf("line %d missing ANSI reset", i)
		}
		// Should contain foreground color escape (38;2;R;G;B)
		if !strings.Contains(line, "\x1b[38;2;") {
			t.Errorf("line %d missing fg color escape", i)
		}
		// Should contain background color escape (48;2;R;G;B)
		if !strings.Contains(line, "\x1b[48;2;") {
			t.Errorf("line %d missing bg color escape", i)
		}
	}
}

func TestRenderHalfBlock_OddHeight(t *testing.T) {
	// 4x3 image → 2 lines (rows 0-1 in line 1, row 2 + transparent in line 2)
	img := image.NewRGBA(image.Rect(0, 0, 4, 3))
	lines := RenderHalfBlock(img, 4)
	if len(lines) != 2 {
		t.Errorf("expected 2 lines for 3px height, got %d", len(lines))
	}
}

func TestRenderHalfBlock_SingleRow(t *testing.T) {
	// 4x1 image → 1 line
	img := image.NewRGBA(image.Rect(0, 0, 4, 1))
	lines := RenderHalfBlock(img, 4)
	if len(lines) != 1 {
		t.Errorf("expected 1 line for 1px height, got %d", len(lines))
	}
}

func TestRenderHalfBlock_ScalesDown(t *testing.T) {
	// 80px wide image, maxCols=40 → output should have 40 characters worth of blocks
	img := image.NewRGBA(image.Rect(0, 0, 80, 4))
	lines := RenderHalfBlock(img, 40)
	if len(lines) == 0 {
		t.Fatal("expected output lines")
	}
	// Count ▄ characters in the first line
	count := strings.Count(lines[0], "▄")
	if count != 40 {
		t.Errorf("expected 40 half-block chars for maxCols=40, got %d", count)
	}
}

func TestRenderHalfBlock_EmptyImage(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 0, 0))
	lines := RenderHalfBlock(img, 40)
	if len(lines) != 0 {
		t.Errorf("expected 0 lines for empty image, got %d", len(lines))
	}
}
