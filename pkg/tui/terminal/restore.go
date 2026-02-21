// ABOUTME: RestoreOnPanic recovers from panics, restores the terminal, and prints the stack trace.
// ABOUTME: Intended for use as a deferred call in the main goroutine.

package terminal

import (
	"fmt"
	"os"
	"runtime/debug"
)

// RestoreOnPanic should be deferred at the top of main (or any
// goroutine that owns the terminal). On panic it restores the cursor,
// exits raw mode via the provided Terminal, prints the panic value
// and stack trace, then exits with code 1.
func RestoreOnPanic(t Terminal) {
	r := recover()
	if r == nil {
		return
	}

	// Best-effort: show cursor and exit raw mode.
	_, _ = os.Stdout.Write([]byte("\033[?25h")) // show cursor
	_ = t.ExitRawMode()

	fmt.Fprintf(os.Stderr, "\npanic: %v\n\n%s\n", r, debug.Stack())
	os.Exit(1)
}

// RecoverGoroutine should be deferred at the top of background goroutines
// that run while the terminal is in raw mode. Unlike RestoreOnPanic it
// does NOT call os.Exit, allowing the main goroutine to handle shutdown.
func RecoverGoroutine(t Terminal) {
	r := recover()
	if r == nil {
		return
	}

	// Best-effort: show cursor and exit raw mode.
	_, _ = os.Stdout.Write([]byte("\033[?25h")) // show cursor
	_ = t.ExitRawMode()

	fmt.Fprintf(os.Stderr, "\ngoroutine panic: %v\n\n%s\n", r, debug.Stack())
}
