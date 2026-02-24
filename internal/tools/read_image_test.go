// ABOUTME: Tests for the read_image tool: valid images, rejection of non-images, size limits
// ABOUTME: Validates path resolution, sandbox enforcement, and error handling

package tools

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/mauromedda/pi-coding-agent-go/internal/agent"
	"github.com/mauromedda/pi-coding-agent-go/internal/permission"
)

func TestReadImage_ValidPNG(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "test.png")
	// Minimal valid PNG: 1x1 pixel
	pngData := minimalPNG()
	if err := os.WriteFile(path, pngData, 0o644); err != nil {
		t.Fatal(err)
	}

	tool := newReadImageTool(nil)
	result, err := tool.Execute(context.Background(), "id1", map[string]any{"path": path}, noopUpdate)
	if err != nil {
		t.Fatal(err)
	}

	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content)
	}
	if len(result.Images) != 1 {
		t.Fatalf("expected 1 image; got %d", len(result.Images))
	}
	if result.Images[0].MimeType != "image/png" {
		t.Errorf("expected image/png; got %s", result.Images[0].MimeType)
	}
	if result.Images[0].Filename != "test.png" {
		t.Errorf("expected filename test.png; got %s", result.Images[0].Filename)
	}
}

func TestReadImage_ValidJPEG(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "photo.jpg")
	// Minimal JPEG header
	jpegData := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46}
	if err := os.WriteFile(path, jpegData, 0o644); err != nil {
		t.Fatal(err)
	}

	tool := newReadImageTool(nil)
	result, err := tool.Execute(context.Background(), "id1", map[string]any{"path": path}, noopUpdate)
	if err != nil {
		t.Fatal(err)
	}

	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content)
	}
	if len(result.Images) != 1 {
		t.Fatalf("expected 1 image; got %d", len(result.Images))
	}
	if result.Images[0].MimeType != "image/jpeg" {
		t.Errorf("expected image/jpeg; got %s", result.Images[0].MimeType)
	}
}

func TestReadImage_NonImageRejected(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "code.go")
	if err := os.WriteFile(path, []byte("package main"), 0o644); err != nil {
		t.Fatal(err)
	}

	tool := newReadImageTool(nil)
	result, err := tool.Execute(context.Background(), "id1", map[string]any{"path": path}, noopUpdate)
	if err != nil {
		t.Fatal(err)
	}

	if !result.IsError {
		t.Fatal("expected error for non-image file")
	}
	if len(result.Images) != 0 {
		t.Errorf("expected 0 images; got %d", len(result.Images))
	}
}

func TestReadImage_TooLarge(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "huge.png")
	// Write file larger than maxImageFileSize
	data := make([]byte, maxImageFileSize+1)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}

	tool := newReadImageTool(nil)
	result, err := tool.Execute(context.Background(), "id1", map[string]any{"path": path}, noopUpdate)
	if err != nil {
		t.Fatal(err)
	}

	if !result.IsError {
		t.Fatal("expected error for oversized image")
	}
}

func TestReadImage_MissingPath(t *testing.T) {
	t.Parallel()

	tool := newReadImageTool(nil)
	result, err := tool.Execute(context.Background(), "id1", map[string]any{}, noopUpdate)
	if err != nil {
		t.Fatal(err)
	}

	if !result.IsError {
		t.Fatal("expected error for missing path")
	}
}

func TestReadImage_FileNotFound(t *testing.T) {
	t.Parallel()

	tool := newReadImageTool(nil)
	result, err := tool.Execute(context.Background(), "id1", map[string]any{"path": "/nonexistent/image.png"}, noopUpdate)
	if err != nil {
		t.Fatal(err)
	}

	if !result.IsError {
		t.Fatal("expected error for missing file")
	}
}

func TestReadImage_SandboxRejected(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	sb, err := permission.NewSandbox([]string{dir})
	if err != nil {
		t.Fatal(err)
	}

	tool := newReadImageTool(sb)
	result, err := tool.Execute(context.Background(), "id1", map[string]any{"path": "/etc/image.png"}, noopUpdate)
	if err != nil {
		t.Fatal(err)
	}

	if !result.IsError {
		t.Fatal("expected error for sandbox violation")
	}
}

func TestReadImage_ToolMetadata(t *testing.T) {
	t.Parallel()

	tool := newReadImageTool(nil)
	if tool.Name != "read_image" {
		t.Errorf("expected name 'read_image'; got %q", tool.Name)
	}
	if !tool.ReadOnly {
		t.Error("expected ReadOnly = true")
	}
}

// noopUpdate discards tool update callbacks.
func noopUpdate(_ agent.ToolUpdate) {}

// minimalPNG returns a valid 1x1 transparent PNG file.
func minimalPNG() []byte {
	return []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, // PNG signature
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52, // IHDR chunk
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, // 1x1
		0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53, // 8-bit RGB
		0xDE, 0x00, 0x00, 0x00, 0x0C, 0x49, 0x44, 0x41, // IDAT chunk
		0x54, 0x08, 0xD7, 0x63, 0xF8, 0xCF, 0xC0, 0x00,
		0x00, 0x00, 0x02, 0x00, 0x01, 0xE2, 0x21, 0xBC,
		0x33, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4E, // IEND chunk
		0x44, 0xAE, 0x42, 0x60, 0x82,
	}
}
