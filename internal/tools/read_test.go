// ABOUTME: Tests for the read tool: normal reads, offset/limit, binary detection, and sandbox
// ABOUTME: Uses t.TempDir for isolated filesystem operations

package tools

import (
	"bytes"
	"context"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mauromedda/pi-coding-agent-go/internal/agent"
	"github.com/mauromedda/pi-coding-agent-go/internal/permission"
)

func TestReadTool_NormalFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	content := "line1\nline2\nline3\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	tool := NewReadTool()
	result, err := tool.Execute(context.Background(), "id1", map[string]any{"path": path}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %s", result.Content)
	}
	if result.Content != content {
		t.Errorf("got %q, want %q", result.Content, content)
	}
}

func TestReadTool_OffsetAndLimit(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "lines.txt")
	content := "line0\nline1\nline2\nline3\nline4\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	tool := NewReadTool()
	result, err := tool.Execute(context.Background(), "id1", map[string]any{
		"path":   path,
		"offset": float64(1), // JSON numbers arrive as float64
		"limit":  float64(2),
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %s", result.Content)
	}

	expected := "line1\nline2\n"
	if result.Content != expected {
		t.Errorf("got %q, want %q", result.Content, expected)
	}
}

func TestReadTool_BinaryDetection(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "binary.bin")
	data := []byte("hello\x00world")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}

	tool := NewReadTool()
	result, err := tool.Execute(context.Background(), "id1", map[string]any{"path": path}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected IsError for binary file")
	}
	if !strings.Contains(result.Content, "binary") {
		t.Errorf("expected 'binary' in error message, got %q", result.Content)
	}
}

func TestReadTool_FileNotFound(t *testing.T) {
	t.Parallel()

	tool := NewReadTool()
	result, err := tool.Execute(context.Background(), "id1", map[string]any{
		"path": "/nonexistent/file.txt",
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected IsError for missing file")
	}
}

func TestReadTool_MissingPath(t *testing.T) {
	t.Parallel()

	tool := NewReadTool()
	result, err := tool.Execute(context.Background(), "id1", map[string]any{}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected IsError for missing path param")
	}
}

func TestReadTool_LargeFileTruncation(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "large.txt")
	// Create content larger than maxReadOutput (100KB)
	content := strings.Repeat("x", maxReadOutput+1000)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	tool := NewReadTool()
	result, err := tool.Execute(context.Background(), "id1", map[string]any{"path": path}, func(_ agent.ToolUpdate) {})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %s", result.Content)
	}
	if !strings.Contains(result.Content, "truncated") {
		t.Error("expected truncation notice in output")
	}
}

func TestReadTool_HugeFileUsesLimitReader(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "huge.txt")

	// Create a file slightly larger than maxFileReadSize (10MB).
	// We write 11MB of 'A' characters.
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	chunk := strings.Repeat("A", 1024*1024) // 1MB
	for range 11 {
		if _, err := f.WriteString(chunk); err != nil {
			f.Close()
			t.Fatal(err)
		}
	}
	f.Close()

	tool := NewReadTool()
	result, err := tool.Execute(context.Background(), "id1", map[string]any{"path": path}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %s", result.Content)
	}
	// The output should be truncated (maxReadOutput = 100KB), but the key
	// point is that the tool did NOT read all 11MB into memory.
	if !strings.Contains(result.Content, "truncated") {
		t.Error("expected truncation notice for huge file")
	}
	// Output must not exceed maxReadOutput + truncation notice.
	if len(result.Content) > maxReadOutput+100 {
		t.Errorf("output too large: %d bytes", len(result.Content))
	}
}

func TestReadTool_ImageFileReturnsImageBlock(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "test.png")

	// Create a minimal valid PNG
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}

	tool := NewReadTool()
	result, err := tool.Execute(context.Background(), "id1", map[string]any{"path": path}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %s", result.Content)
	}
	if !strings.Contains(result.Content, "test.png") {
		t.Errorf("expected filename in content, got %q", result.Content)
	}
	if !strings.Contains(result.Content, "image/png") {
		t.Errorf("expected mime type in content, got %q", result.Content)
	}
	if len(result.Images) != 1 {
		t.Fatalf("expected 1 image block, got %d", len(result.Images))
	}
	if result.Images[0].MimeType != "image/png" {
		t.Errorf("expected image/png, got %q", result.Images[0].MimeType)
	}
	if result.Images[0].Filename != "test.png" {
		t.Errorf("expected test.png, got %q", result.Images[0].Filename)
	}
}

func TestReadTool_NonImageBinaryStillErrors(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "data.bin")
	data := []byte("hello\x00world")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}

	tool := NewReadTool()
	result, err := tool.Execute(context.Background(), "id1", map[string]any{"path": path}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected IsError for non-image binary file")
	}
	if !strings.Contains(result.Content, "binary") {
		t.Errorf("expected 'binary' in error message, got %q", result.Content)
	}
}

func TestReadTool_ImageTooLarge(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "huge.png")

	// Create a PNG header followed by enough bytes to exceed maxImageFileSize
	// We need a valid binary detection (has null bytes) and .png extension
	data := make([]byte, maxImageFileSize+100)
	copy(data, []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1A, '\n'}) // PNG signature
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}

	tool := NewReadTool()
	result, err := tool.Execute(context.Background(), "id1", map[string]any{"path": path}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected IsError for oversized image")
	}
	if !strings.Contains(result.Content, "too large") {
		t.Errorf("expected 'too large' in error, got %q", result.Content)
	}
}

func TestImageExtMIME(t *testing.T) {
	tests := []struct {
		path string
		mime string
		ok   bool
	}{
		{"/foo/bar.png", "image/png", true},
		{"/foo/bar.PNG", "image/png", true},
		{"/foo/bar.jpg", "image/jpeg", true},
		{"/foo/bar.jpeg", "image/jpeg", true},
		{"/foo/bar.gif", "image/gif", true},
		{"/foo/bar.webp", "image/webp", true},
		{"/foo/bar.txt", "", false},
		{"/foo/bar.bin", "", false},
	}
	for _, tt := range tests {
		mime, ok := imageExtMIME(tt.path)
		if ok != tt.ok || mime != tt.mime {
			t.Errorf("imageExtMIME(%q) = (%q, %v), want (%q, %v)", tt.path, mime, ok, tt.mime, tt.ok)
		}
	}
}

func TestReadTool_OutOfSandboxRejected(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	sb, err := permission.NewSandbox([]string{dir})
	if err != nil {
		t.Fatal(err)
	}

	tool := NewReadToolWithSandbox(sb)
	result, err := tool.Execute(context.Background(), "id1", map[string]any{
		"path": "/etc/passwd",
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected IsError for out-of-sandbox path")
	}
	if !strings.Contains(result.Content, "outside allowed") {
		t.Errorf("expected sandbox rejection message, got %q", result.Content)
	}
}
