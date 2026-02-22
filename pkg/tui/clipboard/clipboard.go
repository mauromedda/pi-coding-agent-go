// ABOUTME: Cross-platform clipboard write using pbcopy (macOS) or xclip (Linux)
// ABOUTME: Pipes text to the platform clipboard command via stdin

package clipboard

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// Write copies text to the system clipboard.
func Write(text string) error {
	cmd, args := clipboardCmd()
	if cmd == "" {
		return fmt.Errorf("clipboard not supported on %s", runtime.GOOS)
	}

	c := exec.Command(cmd, args...)
	c.Stdin = strings.NewReader(text)
	return c.Run()
}

// clipboardCmd returns the clipboard command and arguments for the current OS.
func clipboardCmd() (string, []string) {
	switch runtime.GOOS {
	case "darwin":
		return "pbcopy", nil
	case "linux":
		return "xclip", []string{"-selection", "clipboard"}
	default:
		return "", nil
	}
}
