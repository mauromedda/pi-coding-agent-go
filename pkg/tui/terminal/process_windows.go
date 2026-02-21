// ABOUTME: Windows stub for ProcessTerminal resize handling.
// ABOUTME: Placeholder; Windows does not use SIGWINCH signals.

//go:build windows

package terminal

// startResizeListener is a no-op on Windows.
// Windows terminal resize detection requires SetConsoleMode and
// ReadConsoleInput, which is left for future implementation.
func (t *ProcessTerminal) startResizeListener() {
	// No-op: Windows resize detection is not yet implemented.
}
