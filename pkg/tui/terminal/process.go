// ABOUTME: ProcessTerminal implements Terminal using os.Stdout and golang.org/x/term.
// ABOUTME: Manages raw mode state and delegates platform-specific resize handling.

package terminal

import (
	"fmt"
	"os"
	"sync"

	"golang.org/x/term"
)

// ProcessTerminal is a real terminal backed by os.Stdout and x/term.
type ProcessTerminal struct {
	mu       sync.Mutex
	oldState *term.State
	resizeFn func(width, height int)
}

// NewProcessTerminal returns a ProcessTerminal ready for use.
func NewProcessTerminal() *ProcessTerminal {
	return &ProcessTerminal{}
}

// EnterRawMode switches stdin to raw mode, saving the previous state.
func (t *ProcessTerminal) EnterRawMode() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	state, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return fmt.Errorf("entering raw mode: %w", err)
	}
	t.oldState = state
	return nil
}

// ExitRawMode restores the terminal to its previous state.
func (t *ProcessTerminal) ExitRawMode() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.oldState == nil {
		return nil
	}
	if err := term.Restore(int(os.Stdin.Fd()), t.oldState); err != nil {
		return fmt.Errorf("exiting raw mode: %w", err)
	}
	t.oldState = nil
	return nil
}

// Size returns the current terminal dimensions.
func (t *ProcessTerminal) Size() (width, height int, err error) {
	w, h, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return 0, 0, fmt.Errorf("getting terminal size: %w", err)
	}
	return w, h, nil
}

// Write sends bytes to os.Stdout.
func (t *ProcessTerminal) Write(p []byte) (int, error) {
	n, err := os.Stdout.Write(p)
	if err != nil {
		return n, fmt.Errorf("writing to stdout: %w", err)
	}
	return n, nil
}

// OnResize registers a callback invoked when the terminal is resized.
// Platform-specific signal handling is set up by startResizeListener.
func (t *ProcessTerminal) OnResize(fn func(width, height int)) {
	t.mu.Lock()
	t.resizeFn = fn
	t.mu.Unlock()

	t.startResizeListener()
}
