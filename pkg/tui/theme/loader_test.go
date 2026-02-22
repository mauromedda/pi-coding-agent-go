// ABOUTME: Tests for JSON theme file loading and validation
// ABOUTME: Covers valid load, missing fields fallback, invalid JSON, and file not found

package theme

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFile_ValidJSON(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "test.json")
	data := `{
		"name": "custom",
		"palette": {
			"primary": "\u001b[97m",
			"secondary": "\u001b[90m",
			"muted": "\u001b[2m",
			"accent": "\u001b[1m",
			"success": "\u001b[32m",
			"warning": "\u001b[33m",
			"error": "\u001b[31m",
			"info": "\u001b[36m",
			"border": "\u001b[90m",
			"selection": "\u001b[7m",
			"prompt": "\u001b[1m",
			"tool_read": "\u001b[36m",
			"tool_bash": "\u001b[33m",
			"tool_write": "\u001b[32m",
			"tool_other": "\u001b[35m",
			"footer_path": "\u001b[1m",
			"footer_branch": "\u001b[36m",
			"footer_model": "\u001b[36m",
			"footer_cost": "\u001b[33m",
			"footer_perm": "\u001b[32m",
			"bold": "\u001b[1m",
			"dim": "\u001b[2m",
			"italic": "\u001b[3m",
			"underline": "\u001b[4m"
		}
	}`
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}

	th, err := LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile() error: %v", err)
	}
	if th.Name != "custom" {
		t.Errorf("Name = %q; want %q", th.Name, "custom")
	}
	if th.Palette.Success.Code() != "\x1b[32m" {
		t.Errorf("Palette.Success.Code() = %q; want %q", th.Palette.Success.Code(), "\x1b[32m")
	}
}

func TestLoadFile_MissingFields_FallsBackToDefault(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "partial.json")
	data := `{
		"name": "partial",
		"palette": {
			"success": "\u001b[32m"
		}
	}`
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}

	th, err := LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile() error: %v", err)
	}
	if th.Name != "partial" {
		t.Errorf("Name = %q; want %q", th.Name, "partial")
	}
	// Explicitly set field
	if th.Palette.Success.Code() != "\x1b[32m" {
		t.Errorf("Success = %q; want %q", th.Palette.Success.Code(), "\x1b[32m")
	}
	// Unset field should fall back to default
	if th.Palette.Error.Code() == "" {
		t.Error("Error should fall back to default, got empty")
	}
}

func TestLoadFile_InvalidJSON(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(path, []byte("{not json"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadFile(path)
	if err == nil {
		t.Error("LoadFile() should return error for invalid JSON")
	}
}

func TestLoadFile_NotFound(t *testing.T) {
	t.Parallel()

	_, err := LoadFile("/nonexistent/theme.json")
	if err == nil {
		t.Error("LoadFile() should return error for missing file")
	}
}
