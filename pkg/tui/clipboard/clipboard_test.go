// ABOUTME: Tests for cross-platform clipboard write operations
// ABOUTME: Verifies command selection for macOS (pbcopy) and Linux (xclip)

package clipboard

import (
	"runtime"
	"testing"
)

func TestClipboardCommand_Darwin(t *testing.T) {
	t.Parallel()

	if runtime.GOOS != "darwin" {
		t.Skip("darwin-only test")
	}

	cmd, args := clipboardCmd()
	if cmd != "pbcopy" {
		t.Errorf("expected pbcopy on darwin, got %q", cmd)
	}
	if len(args) != 0 {
		t.Errorf("expected no args for pbcopy, got %v", args)
	}
}

func TestClipboardCommand_Linux(t *testing.T) {
	t.Parallel()

	if runtime.GOOS != "linux" {
		t.Skip("linux-only test")
	}

	cmd, args := clipboardCmd()
	if cmd != "xclip" {
		t.Errorf("expected xclip on linux, got %q", cmd)
	}
	if len(args) != 2 || args[0] != "-selection" || args[1] != "clipboard" {
		t.Errorf("expected [-selection clipboard] for xclip, got %v", args)
	}
}

func TestWrite_EmptyString(t *testing.T) {
	t.Parallel()

	// Writing empty string should not error
	err := Write("")
	if err != nil {
		t.Errorf("Write(\"\") returned unexpected error: %v", err)
	}
}
