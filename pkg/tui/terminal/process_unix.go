// ABOUTME: Unix-specific SIGWINCH handling for ProcessTerminal resize events.
// ABOUTME: Spawns a goroutine that listens for SIGWINCH and invokes the resize callback.

//go:build unix

package terminal

import (
	"os"
	"os/signal"
	"syscall"
)

// startResizeListener sets up a SIGWINCH handler that calls the
// resize callback with the new terminal dimensions.
func (t *ProcessTerminal) startResizeListener() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGWINCH)

	go func() {
		for range sigCh {
			t.mu.Lock()
			fn := t.resizeFn
			t.mu.Unlock()

			if fn == nil {
				continue
			}

			w, h, err := t.Size()
			if err != nil {
				continue
			}
			fn(w, h)
		}
	}()
}
