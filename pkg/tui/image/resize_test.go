// ABOUTME: Tests for the smart image resize pipeline
// ABOUTME: Verifies dimension capping, format fallback, and size reduction

package image

import (
	"testing"
)

func TestResize_AlreadySmall(t *testing.T) {
	data := makePNG(t, 100, 100)
	out, dim, mime, err := Resize(data, 2000, 5*1024*1024)
	if err != nil {
		t.Fatal(err)
	}
	if dim.Width != 100 || dim.Height != 100 {
		t.Errorf("got %dx%d, want 100x100", dim.Width, dim.Height)
	}
	if mime != "image/png" {
		t.Errorf("got mime %q, want image/png", mime)
	}
	// Should return original data unchanged
	if len(out) != len(data) {
		t.Errorf("expected same size, got %d vs %d", len(out), len(data))
	}
}

func TestResize_DimensionCap(t *testing.T) {
	data := makePNG(t, 400, 200)
	// Cap at 100 max dimension
	_, dim, _, err := Resize(data, 100, 5*1024*1024)
	if err != nil {
		t.Fatal(err)
	}
	if dim.Width > 100 || dim.Height > 100 {
		t.Errorf("expected dimensions <= 100, got %dx%d", dim.Width, dim.Height)
	}
	// Aspect ratio preserved: 400x200 scaled to 100 max → 100x50
	if dim.Width != 100 || dim.Height != 50 {
		t.Errorf("got %dx%d, want 100x50 (aspect ratio)", dim.Width, dim.Height)
	}
}

func TestResize_ReturnsValidImage(t *testing.T) {
	data := makePNG(t, 300, 300)
	out, _, mime, err := Resize(data, 100, 5*1024*1024)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) == 0 {
		t.Error("expected non-empty output")
	}
	// MIME should be png or jpeg
	if mime != "image/png" && mime != "image/jpeg" {
		t.Errorf("unexpected mime %q", mime)
	}
	// Verify we can read dimensions from the output
	dim, err := GetDimensions(out)
	if err != nil {
		t.Fatalf("GetDimensions on resized output: %v", err)
	}
	if dim.Width > 100 || dim.Height > 100 {
		t.Errorf("resized output exceeds max dim: %dx%d", dim.Width, dim.Height)
	}
}

func TestResize_EmptyInput(t *testing.T) {
	_, _, _, err := Resize(nil, 100, 5*1024*1024)
	if err == nil {
		t.Error("expected error for nil input")
	}
}

func TestResize_InvalidImage(t *testing.T) {
	_, _, _, err := Resize([]byte("not an image"), 100, 5*1024*1024)
	if err == nil {
		t.Error("expected error for invalid image data")
	}
}

func TestResize_LargeImageGetsSmaller(t *testing.T) {
	data := makePNG(t, 1000, 1000)
	out, dim, _, err := Resize(data, 200, 5*1024*1024)
	if err != nil {
		t.Fatal(err)
	}
	if dim.Width != 200 || dim.Height != 200 {
		t.Errorf("got %dx%d, want 200x200", dim.Width, dim.Height)
	}
	// Output should be smaller than original
	if len(out) >= len(data) {
		t.Errorf("expected smaller output: %d >= %d", len(out), len(data))
	}
}
