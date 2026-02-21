// ABOUTME: VirtualTerminal implements Terminal for testing without a real TTY.
// ABOUTME: Captures output in a buffer and tracks raw-mode enter/exit calls.

package terminal

import (
	"bytes"
	"fmt"
	"sync"
)

// VirtualTerminal is a fake Terminal for unit tests.
// It records written output and tracks raw-mode transitions.
type VirtualTerminal struct {
	mu        sync.Mutex
	buf       bytes.Buffer
	width     int
	height    int
	rawMode   bool
	resizeFn  func(width, height int)
	enterCount int
	exitCount  int
}

// NewVirtualTerminal returns a VirtualTerminal with the given dimensions.
func NewVirtualTerminal(width, height int) *VirtualTerminal {
	return &VirtualTerminal{
		width:  width,
		height: height,
	}
}

// EnterRawMode records a raw-mode entry.
func (v *VirtualTerminal) EnterRawMode() error {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.rawMode = true
	v.enterCount++
	return nil
}

// ExitRawMode records a raw-mode exit.
func (v *VirtualTerminal) ExitRawMode() error {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.rawMode = false
	v.exitCount++
	return nil
}

// Size returns the configured terminal dimensions.
func (v *VirtualTerminal) Size() (width, height int, err error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	return v.width, v.height, nil
}

// Write appends data to the internal buffer.
func (v *VirtualTerminal) Write(p []byte) (int, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	n, err := v.buf.Write(p)
	if err != nil {
		return n, fmt.Errorf("writing to virtual buffer: %w", err)
	}
	return n, nil
}

// OnResize stores the resize callback.
func (v *VirtualTerminal) OnResize(fn func(width, height int)) {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.resizeFn = fn
}

// --- Test helpers (not part of Terminal interface) ---

// Output returns everything written so far.
func (v *VirtualTerminal) Output() string {
	v.mu.Lock()
	defer v.mu.Unlock()

	return v.buf.String()
}

// Reset clears the output buffer.
func (v *VirtualTerminal) Reset() {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.buf.Reset()
}

// IsRawMode reports whether raw mode is currently active.
func (v *VirtualTerminal) IsRawMode() bool {
	v.mu.Lock()
	defer v.mu.Unlock()

	return v.rawMode
}

// EnterCount returns how many times EnterRawMode was called.
func (v *VirtualTerminal) EnterCount() int {
	v.mu.Lock()
	defer v.mu.Unlock()

	return v.enterCount
}

// ExitCount returns how many times ExitRawMode was called.
func (v *VirtualTerminal) ExitCount() int {
	v.mu.Lock()
	defer v.mu.Unlock()

	return v.exitCount
}

// SetSize updates the terminal dimensions and, if a resize callback
// is registered, invokes it with the new size.
func (v *VirtualTerminal) SetSize(width, height int) {
	v.mu.Lock()
	v.width = width
	v.height = height
	fn := v.resizeFn
	v.mu.Unlock()

	if fn != nil {
		fn(width, height)
	}
}
